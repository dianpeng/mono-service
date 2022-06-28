fn test1() {
  assert::yes("a" ~ r"a");
  assert::yes("ab" ~ r"ab");
  assert::yes("abcd" ~ r"a.*");
  assert::no("abcd" ~ r"cc");
  assert::no("abcd" ~ r"^cd");
  assert::yes("abcd" !~ r"^cd");
  assert::yes("xxx" !~ r"cd");

  {
    let re = r"xx";
    assert::yes("bb" !~ re);
  }
}

test {
  test1();
  assert::eq(type(r""), "regexp");
}
