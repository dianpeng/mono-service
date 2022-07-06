global {
  data = rand::str(1024*512);
}

config service {
  .name = "512k";
  .router = "[GET]/512k";

  application noop();
  response event("response");
}

rule response {
  response.status = 200;
  response.body = data;
}
