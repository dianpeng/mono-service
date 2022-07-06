global {
  data = rand::str(1024*64);
}

config service {
  .name = "64k";
  .router = "[GET]/64k";

  application noop();
  response event("response");
}


rule response {
  response.status = 200;
  response.body = data;
}
