module zz

global {
  mod2_global_a = 10;
  mod2_global_b = 20;
}

session {
  mod2_session_a = 10;
  mod2_session_b = 20;
}

fn Add(a, b) {
  return a + b;
}
