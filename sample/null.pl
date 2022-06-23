config service {
  .name = "null";
  .router = "[GET,POST]/xxx";

  application noop();

  response {
    .event("xxx");
    .event("yyy");
  }
}

session {
  a = 10;
  b = 20;
  c = 200 - 1 + 1;
  ALOHA=0;
}

fn HelloWorld() {
  return "hello world";
}

rule xxx => {
  response.status = 302;
  response.body = "Hello World";
}

rule yyy => {
  let data = response.body.stream:cacheString();
  println("response: ", data);
}

rule response => {
  let proxy_url = request.header["x-proxy-url"];
  let list_of_url = str::split(proxy_url, ';');
  body => "{{c:to_string()}}\n{{HelloWorld()}}\n\n\n";
  status => c;
  println("Hello World", c, a, b);
}

rule error => {
  dprint(phase, error);
}

rule "response.interceptor.status" {
  println("===================");
}

rule "response.interceptor.body" {
  println("===================");
  emit aloha;
  println("after aloha");
  println("===================");
}

rule "aloha" {
  if ALOHA < 10 {
    println("ALOHA");
    ALOHA++;
    emit aloha;
  }
}
