config redis_vhost {
  .name = "redis_test";
  .listener = "test";
}

rule "redis.HGET" {
  println("command info: (", $.command, ", ", $.length, ", ", $.category, ")");
  conn:writeString("Always HGET yeah!");
}

rule "redis.*" {
  println("command info: (", $.command, ", ", $.length, ", ", $.category, ")");
  conn:writeString("Hello World");
}
