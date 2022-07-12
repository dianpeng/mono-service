config service {
  .name = "resp";
  .router = "[GET]/yyy";

  application noop();

  response response(
    200,
    "Hello World",
    {
      "a" : "b",
      "c" : "d"
    },
    true
  );
}
