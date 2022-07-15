global {
  data = rand::str(1024*256);
}

config service {
  .name = "256k";
  .router = "[GET]/256k";

  application noop();
  response event("response");
}

rule response {
  response.status = 200;
  response.body = data;
}
