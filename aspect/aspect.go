package aspect

import (
	"reflect"
	"xianhetian.com/framework/logger"
)

type AdviceType int

const (
	AdviceBefore = iota
	AdviceAfter
	AdviceAfterReturning
)

// 切面
type Aspect struct {
	advices []Advice
}

// 通知
type Advice struct {
	Method reflect.Method
	Type   AdviceType
}

// 新建通知
func (a *Aspect) AddAdvice(adviceFunc interface{}, adviceType AdviceType) {
	if adviceFunc == nil {
		logger.Error("无法创建通知：通知方法为空")
		return
	} else if adviceType != AdviceBefore && adviceType != AdviceAfter && adviceType != AdviceAfterReturning {
		logger.Error("无法创建通知：通知类型不合法")
		return
	} else {
		for _, advice := range a.advices {
			if adviceType == advice.Type && reflect.ValueOf(adviceFunc) == advice.Method.Func {
				logger.Error("无法创建通知： 通知方法已经加入到该通知类型中")
				return
			}
		}
		a.advices = append(a.advices, Advice{
			Method: reflect.Method{Func: reflect.ValueOf(adviceFunc), Type: reflect.TypeOf(adviceFunc)},
			Type:   adviceType,
		})
	}
}

// 增加切入点
func (a *Aspect) AddPointcut(methodName string, adviceType AdviceType, structs, pointcut interface{}) {
	fn := a.addPointcutWorker(methodName, adviceType, structs)
	a.join(pointcut, fn)
}

// 移除通知
func (a *Aspect) RemoveAdvice(adviceFunc interface{}, adviceType AdviceType) {
	i := a.GetAdviceIndex(adviceFunc, adviceType)
	if i != -1 {
		a.advices = append(a.advices[:i], a.advices[i+1:]...)
		return
	}
	logger.Error("无法删除通知：通知不存在")
	return
}

// 获得通知下标
func (a *Aspect) GetAdviceIndex(adviceFunc interface{}, adviceType AdviceType) (index int) {
	for i, advice := range a.advices {
		if reflect.ValueOf(adviceFunc) == advice.Method.Func && adviceType == advice.Type {
			return i
		}
	}
	return -1
}

func (a *Aspect) addPointcutWorker(methodName string, adviceType AdviceType, structs interface{}) (fn func(args []reflect.Value) []reflect.Value) {
	m := reflect.ValueOf(structs).MethodByName(methodName)
	if !reflect.Value.IsValid(m) {
		logger.Debug("没有方法匹配名称%s；%T", methodName, structs)
		return nil
	}
	adviceFunc := func() {
		for j, advice := range a.advices {
			if advice.Type == adviceType {
				a.advices[j].Method.Func.Call(nil)
			}
		}
	}
	switch adviceType {
	case AdviceBefore:
		fn = func(args []reflect.Value) []reflect.Value {
			adviceFunc()
			return m.Call(args)
		}
	case AdviceAfter:
		fn = func(args []reflect.Value) []reflect.Value {
			returnValues := m.Call(args)
			adviceFunc()
			return returnValues
		}
	case AdviceAfterReturning:
		fn = func(args []reflect.Value) []reflect.Value {
			returnValues := m.Call(args)
			for idx := 0; idx < len(returnValues); idx++ {
				if returnValues[idx].Type() == reflect.TypeOf((*error)(nil)).Elem() && !returnValues[idx].IsNil() {
					return returnValues
				}
			}
			adviceFunc()
			return returnValues
		}
	}
	return
}

func (a *Aspect) join(pointcut interface{}, fn func([]reflect.Value) []reflect.Value) {
	f := reflect.ValueOf(pointcut).Elem()
	f.Set(reflect.MakeFunc(f.Type(), fn))
}
