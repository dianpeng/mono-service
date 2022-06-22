// The map is unordered, but our implementation currently does have some
// order, the testing is very implicit and should be in sync with the
// internal implementation
fn map_iter() {
  let m = {
    "key" : 1,
    "val" : 2
  };

  let i = 0;
  let keys = [];
  let vals = [];

  for let k, v = m {
    keys:push_back(k);
    vals:push_back(v);
  }

  assert::eq(keys:length(), 2);
  assert::eq(vals:length(), 2);
  assert::eq(keys[0], "key");
  assert::eq(keys[1], "val");
  assert::eq(vals[0], 1);
  assert::eq(vals[1], 2);

  let v = |{
    let i = 0;
    for let _, k = [] {
      i++;
    }
    i;
  };

  assert::eq(v, 0);
}

fn list_iter() {
  assert::eq(|{
    let i = 0;
    for let _, _ = [] {
      i++;
    }
    i;
  }, 0);

  assert::eq(|{
    let i = 0;
    for let _, _ = [1] {
      i++;
    }
    i;
  }, 1);

  assert::eq(|{
    let i = 0;
    for let _, _ = [2] {
      i++;
    }
    i;
  }, 1);
}

fn str_iter() {
  assert::eq(|{
    let i = 0;
    for let _, _ = "" {
      i++;
    }
    i;
  }, 0);

  assert::eq(|{
    let i = 0;
    for let _, _ = "a" {
      i++;
    }
    i;
  }, 1);

  assert::eq(|{
    let i = 0;
    for let _, _ = "ab" {
      i++;
    }
    i;
  }, 2);
}

fn pair_iter() {
  assert::eq(if true {
    let i = 0;
    for let _, _ = ("a", "b") {
      i++;
    }
    i;
  }, 2);
}

test {
  map_iter();
  list_iter();
  str_iter();
  pair_iter();
}
