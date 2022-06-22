package framework

var responsemap middlewarefactorymap = newmiddlewarefactorymap()

func AddResponseFactory(
	name string,
	f MiddlewareFactory,
) {
	responsemap.add(name, f)
}

func GetResponseFactory(name string) MiddlewareFactory {
	return responsemap.get(name)
}
