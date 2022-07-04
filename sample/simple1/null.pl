global {
  x = http::get("https://www.tmall.com").body:string();
}

session {
  a = 10;
  b = 20;
  c = 200 - 1 + 1;
  ALOHA=0;
}

config service {
  .name = "null";
  .router = "[GET,POST]/xxx";

  application event(
    "application",
    "this is the value"
  );

  response event(
    "response"
  );
}

fn HelloWorld() {
  return "hello world";
}

rule application {
}

rule response {
  let resp = http::get("https://www.tmall.com");
  response.status = resp.status;
  response.header:set("server", resp.header.server);
  response.header:set("via", resp.header.via);
  let i = 0;
  for let _, _ = resp.header {
    i++;
  }
  assert::yes(i >= 10);
  response.body = resp.body;
}
