config redis_vhost {
  .name = "redis_test";
  .listener = "test";
}

rule "redis.HGET" {
  println("command info: (", $.command, ", ", $.length, ", ", $.category, ")");
  if $.length != 1 {
    conn:writeString("invalid command!");
  } else {
    let value = $:asString(0);
    println("command HGET ", value);
    // write the response back
    conn:writeString("Always HGET yeah!");
  }
}

rule "redis.*" {
  println("command info: (", $.command, ", ", $.length, ", ", $.category, ")");
  conn:writeString("Hello World");
}
