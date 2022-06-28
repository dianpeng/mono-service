fn test1() {
  assert::eq([]:length(), 0);
  assert::eq([]:push_back(1):length(), 1);
  assert::eq(
    []:push_back(1):push_back(2):push_back(3),
    [1, 2, 3]
  );
  assert::eq(
    []:push_back(1):pop_back():push_back(2):pop_back(),
    []
  );
  assert::eq([]:extend([1, 2, 3]):length(), 3);

  assert::eq(
    // Using block expression statement
    |{
      let v = [];
      v:push_back(1);
      v:extend([2, 3]);
      v:push_back(3);
      v:pop_back();
      v:length();
    },
    3
  );
}

fn test2() {
  assert::eq(
    |{
      let i = 0;
      for let index, value = [] {
        i++;
      }
      i;
    },
    0
  );

  assert::eq(
    |{
      let i = 0;
      for let _, _ = [1] {
        i++;
      }
      i;
    },
    1
  );

  let sum = fn(list) {
    let tt = 0;
    for let _, v = list {
      tt += v;
    }
    return tt;
  };

  assert::eq([1, 2, 3, 4] | sum, 1+2+3+4);
  assert::eq([] | sum, 0);
  assert::eq([1] | sum, 1);
}

fn test3() {
  assert::eq([]:slice(1, 100), []);
  assert::eq([1, 2, 3]:slice(1), [2, 3]);
  assert::eq([1, 2, 3]:slice(0, 2), [1, 2]);
  assert::eq([1, 2, 3]:slice(100, 200), []);
  assert::eq([1, 2, 3]:slice(1, 100), [2, 3]);
  assert::eq([1, 2, 3]:slice(100, 3), []);
}

fn testBasic() {
  assert::eq(type([]), "list");
}

fn testNest() {
  assert::eq(
    [
      [
        [
          [
            [
              [
                [
                  [
                    [
                      [10]
                    ]
                  ]
                ]
              ]
            ]
          ]
        ]
      ]
    ][0][0][0][0][0][0][0][0][0][0], 10);
}


test {
  test1();
  test2();
  test3();
  testBasic();
  testNest();
}
