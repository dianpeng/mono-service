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

  request {
    // adding a request header
    .header_add(("a", "b"));

    // adding yet another request header
    .header_add(("a", "c"));

    // adding yet another request header yeah!
    .header_add(("a", "d"));

    // okay, just throw all already added header away and set another header
    .header_set(("a", "EEE"));

    // lets emit an event to see what are supposed to be happening here
    .event("show");
  }

  application event(
    "application",
    "this is the value"
  );

  response random(200);
}

fn HelloWorld() {
  return "hello world";
}

rule show {
  for let k, v = request.header {
    println("req: ", k, " => ", v);
  }
  println("access log: ", log.format);

  println("================= appendix");
  for let k, v = log.appendix {
    println(k, " => ", v);
  }

  println("================= appendix");
  log.appendix:push_back("a");
  for let k, v = log.appendix {
    println(k, " => ", v);
  }
}

rule application {
  println("the application is been kicked in: {{$}}");
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
