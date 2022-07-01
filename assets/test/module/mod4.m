module mod4

global {
  mod4_global = 100;
}

session {
  mod4_session = "true";
}

iter bIter() {
  for let i = 0; i < 10; i++ {
    yield (i, i);
  }
}

iter aIter(map) {
  for let k, v = map {
    yield (k, v);
  }
}
