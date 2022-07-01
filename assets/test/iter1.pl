fn testIter() {
  let x = [];
  for let i, v = iter xxx() {
    x:push_back(i);
  }
  assert::eq(x, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]);

  let yy = [];
  for let i, v = iter uuu() {
    yy:push_back(i);
  }
  assert::eq(yy, [0, 1, 2, 3, 4]);
}

test {
  testIter();
}

fn zzz() {
  return try yyy else 1000;
}

iter xxx() {
  for let i = 0; i < 10; i++ {
    yield (i, "vvv");
  }
  zzz();
}

iter yyy() {
  for let i = 0; i < 5; i++ {
    yield(i, "vvv");
    zzz();
  }
}

iter uuu() {
  for let i, _ = iter yyy() {
    yield (i, 'xxx');
  }
}
