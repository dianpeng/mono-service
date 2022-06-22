package framework

// used for request direction middleware, ie for authentication etc ...
var requestmap middlewarefactorymap = newmiddlewarefactorymap()

func AddRequestFactory(
	name string,
	f MiddlewareFactory,
) {
	requestmap.add(name, f)
}

func GetRequestFactory(name string) MiddlewareFactory {
	return requestmap.get(name)
}
