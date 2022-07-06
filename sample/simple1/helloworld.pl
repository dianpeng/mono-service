config service {
  .name = "hello_world";
  .router = "[GET]/helloworld";
  application noop();
  response event("response");
}

rule response {
  response.status = 200;
  response.body = "Hello World";
}
