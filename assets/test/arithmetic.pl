// basic arithmetic testing, the entry is test for now

fn testBasicArithmetic() {
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
  assert::eq([], []);
  assert::eq([1], [1]);

  {
    let aa;
    assert::eq(null, aa);
  }

  assert::eq({}, {});
  assert::eq({
    "a": 10,
    "b": 20
  },
  {
    "a": 1+8+1,
    "b": 22-2
  });
}

fn testScopingRules() {
  {
    {
      let v = 123;
      assert::eq(v, 1+100+22);
    }
  }

  {{{{{{{{{{}}}}}}}}}}
  {{{{{{{{{{{{{{{{{{{{{{{{{{{{ assert::eq(true, true); }}}}}}}}}}}}}}}}}}}}}}}}}}}}
  {
    let a0 = 50;
    {
      let b0 = 40;
      {
        let c0 = 5;
        {
          let d0 = 5;
          {
            let e0 = 100;
            assert::eq(e0, a0 + b0 + c0 + d0);
          }
        }
      }
    }
  }
  {
    let a = 10;
    {
      let a = 20;
      {
        let a = 30;
        assert::eq(a, 30);
      }
      assert::eq(a, 20);
    }
    assert::eq(a, 10);
  }
}

// entry
rule test {
  testBasicArithmetic();
  testScopingRules();
}
