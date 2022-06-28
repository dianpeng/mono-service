fn myfoo() {
}

fn test1() {
  let xx = myfoo;
  assert::eq(type(xx), "closure");
  assert::throw(
    fn() {
      // this should throw an exception since the myfoo is a function symbol
      // which is not candidate for mutation, so it will try to lookup the
      // symbol via dynamic variable which will generate an error
      myfoo = 10;
    }
  );
}

test {
  test1();
}
