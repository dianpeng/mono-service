fn testStr1() {
  assert::eq("":to_string(), "");
  assert::eq("":length(), 0);
  assert::eq("a":length(), 1);
  assert::eq("a":to_upper(), "A");
  assert::eq("A":to_lower(), "a");
  assert::eq("aabb":substr(1, 2), "a");
  assert::eq("aabb":substr(1), "abb");
  assert::eq("aabb":substr(1, 10000), "abb");
}

fn testStrCon() {
  assert::eq("" + "a", "a");
  assert::eq("a" + "b", "ab");
  assert::eq(("a" + "b"):to_upper(), "AB");
  assert::eq(("A" + "B"):to_lower(), "ab");
}

fn testStrIndex() {
  assert::eq("a":index("a"), 0);
  assert::eq("abc":index("c"), 2);
  assert::eq("a":index("c"), -1);

  // from index
  assert::eq("abcabc":index("c", 3), 5);
}

fn testStrInter1() {
  {
    let x = 10;
    assert::eq("a{{x}}b", "a10b");
  }
  {
    assert::eq("a{{'hello world'}}b", "ahello worldb");
  }
}

fn testStrIndex2() {
  assert::eq("a"[0], "a");
  assert::eq("ab"[1], "b");
}

test {
  testStr1();
  testStrCon();
  testStrIndex();
  testStrInter1();
  testStrIndex2();
  assert::eq(type(""), "string");
}
