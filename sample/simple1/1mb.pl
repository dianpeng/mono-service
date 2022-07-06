global {
  data = rand::str(1024*1024);
}

config service {
  .name = "1mb";
  .router = "[GET]/1mb";

  application noop();
  response event("response");
}

rule response {
  response.status = 200;
  response.body = data;
}
