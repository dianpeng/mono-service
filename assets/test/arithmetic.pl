// basic arithmetic testing, the entry is test for now

test {
  assert::eq(1, 1, "done");
  assert::eq(1+1, 2);
  assert::eq(1*2, 2);
  assert::eq(1+1*2, 3);
  assert::eq(1+1/1, 2);
  assert::eq(1*2+1, 3);
  assert::yes(1>=1);
  assert::yes(1<=1);
  assert::yes(1==1);
  assert::yes(1>=0+1);
  assert::yes(1<1+1);

  assert::no(1 > 1);
  assert::no(1 < 1);
  assert::no(1 >= 2);
  assert::no(1 <= 0);
  assert::no(1 != 1);
  {
    assert::yes(1<1+1);
    ;;;
  }
  {}
}
