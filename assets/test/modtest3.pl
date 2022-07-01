import (
  "assets/test/module/mod4.m"
)

fn test1() {
  let mm = {
    "a": "b",
    "c": "d"
  };
  let um = {};
  for let k, v = iter mod4::aIter(mm) {
    um:set(k, v);
  }
  assert::eq(um, mm);
}

fn test2() {
  let list = [];
  for let k, _ = iter mod4::bIter() {
    list:push_back(k);
  }
  assert::eq(list, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]);
}

test {
  test1();
  test2();
}
