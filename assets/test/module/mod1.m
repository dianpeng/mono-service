module mm

global {
  mod1_global_a = 10;
  mod1_global_b = 20;
}

session {
  mod1_session_a = 10;
  mod1_session_b = 20;
}

fn yy() {
  return "Hello World";
}
