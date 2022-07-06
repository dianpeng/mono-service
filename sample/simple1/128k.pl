global {
  data = rand::str(1024*128);
}

config service {
  .name = "128k";
  .router = "[GET]/128k";

  application noop();
  response event("response");
}


rule response {
  response.status = 200;
  response.body = data;
}
