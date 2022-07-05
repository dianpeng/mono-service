# PL (策略语言)

## 简介

PL是一个专门为Web/Http服务器可编程设计的脚本语言。通常来说，软件的复杂度到达一定程度的时候，总会需要提供可编程能力。Web/Http服务器常常如此。常见的Web服务器，如Nginx/Envoy等，常常通过嵌入Lua，或者其他的通用编程语言来提供一定的可编程定制化能力。但是通用的脚本语言，往往在设计初衷上并不是为Http/Web服务器而设计的。PL则不同，他是从一开始就是为Web/Http服务器的定制化逻辑设计。PL采用被动事件触发的方式，在串联Web/Http服务器各个模块或者组建的的时候侵入性更弱，更加灵活。PL是笔者根据多年开发工业界Web/Http服务器的反思而设计的。


## 特性

1. 动态类型
	1. 丰富类型系统

2. 适配Web/Http场景的功能
	1. 正则表达式Literal
	2. 模版Literal，可扩展
		1. 内置Go/Pongo2/Markdown模版渲染
	3. 字符串模版
	4. JSON
	5. 大量的builtin库
		1. 用户可以利用Go语言快速灵活的扩充运行时功能
	6. 事件触发
	7. 语言内置配置
		1. 不需要类似大部分Web/Http服务器，在一个JSON/XML/YAML配置中内联代码
	7. 语言变量支持Http Session生命周期
		1. 变量随着Http请求Session开始重新求值

3. 丰富的语言表达能力
	1. 支持可变，不变变量
	2. 闭包，高阶函数，上值自动捕获
	3. 块求值
		1. if
		2. try
		3. block
	4. 丰富的控制流
		1. 条件
		2. 循环
		3. 异常
		4. 函数
		5. 事件规则
	5. 模块支持


## 语言

### 类型


* 基本类型
  1. 整数
  2. 浮点数
  3. Null
  4. 布尔值
  5. 字符串

* 聚合类型
  1. 数组
  2. Map
  3. 正则表达式
  4. Pair

* 其他类型
  1. 迭代器
  2. 闭包
  3. 用户扩展类型 (用于支持Go的类型扩展)


### 基本执行单元

PL支持如下几种执行单元

1. 函数
2. 规则
3. 全局静态变量块
4. Session变量块
5. 全局配置块


1. Function

和大部分动态语言一样，PL允许将statement组织成函数个体。PL支持，全局的命名函数和匿名函数。匿名函数允许捕获上值形成闭包。全局命名函数全局可见，可以被代码的不同部分调用。

```

fn foo() {
  let v = 100;
  return fn() {
	return fn() {
    	return v; // 内部的两个嵌套匿名函数，捕获上值v
    };
  };
};

```

2. 规则

规则和函数相似，也是组织功能的单元。不同的是，规则有个事件名称和一个可选的事件上下文。规则内部允许使用一种叫做action语句返回值。action语句不会中断规则执行，规则内部可以多次发出action语句用于和运行时交流。整个服务器的各种流程连接主要是用规则。比如，设置的middleware执行过程中会去发出事件，来触发用户定义的规则。规则使得整个服务形成了一个松耦合状态；同时可以轻松的定制化请求回复流程的各个阶段，并且规则可以同步，也可以移步执行。


```

// 规则a，规则的触发事件为“a"
rule a {
}

// 事件名称为任意字符串，不一定是合法ID，因此规则也可以用字符串来定义
rule "b" {
}

// 或者用[]定义起来，这个语法纯粹是为了方便阅读
rule [c] {
}

// 用户可以直接省略规则关键字rule
cc {
}

// 在规则代码块内，除了可以使用常见的控制流，语句，表达式，还允许action语句。action语句的左边为一个ID，表示action的名称，然后用箭头=>
// 表示赋值。他类似赋值表达式，但是左边不是变量而是action名称，该语句实际功能类似return，只不过该语句不中断规则执行。规则从头执行到尾，
// 规则内部不允许return
// 用户也可以用"$"来引用当前规则的上下文，如果规则的上下文为空，那么$的值为null

rule xx {
  let some_context_value = $.some_value;
  status => 200; // action语句，将status设置为200
  body => "Hello World"; // action语句，将body设置为Hello World

  foo();  // 调用其他函数
}

```

用户可以简单的认为，规则是个规定了函数签名的特殊函数，规则上下文$提供了函数的入参，而action提供了多次返回的规则返回值。为什么这么是的原因是，规则允许更为方便的书写和管理，并且函数签名的规定导致各个部分的串联更加容易。另外，规则的引入并不保证规则的立即执行，在某些情况下规则可以在背景异步执行，函数不必等待规则的返回，规则没有返回值。



3. 全局块

全局静态块是一个全局唯一的静态变量定义。该块在服务器启动的时候进行求值，然后变量的值固定不变。注意，用户任然可以改变某个值的内部情况，比如Map的元素等。但是，全局静态区域的变量绑定的对应值不会发生变化。注意，全局区域必须定义在整个代码的最前面。


```
global {
  const1 = http::get("https://www.tmall.com").body:string(); // 存储tmall首页HTML到string
  const2 = 100;
  const3 = [];

  // 调用更为复杂的全局初始化函数，可以这么做，但是不推荐
  _ = myGlobalInit(); // _表示忽略该变量，_ 可以重复定义
  _ = (fn() {
    return "Nothing Special";
  })();
}

func myGlobalInit() {
}

```

4. Session 变量块

Session变量块定义的变量是关联每次HTTP请求session，每次HTTP请求进入，Session内部的变量会重新求值，Session的变量被某次HTTP请求独占。注意，session必须定义在所有的rule之前，global区域之后，否则语法错误。

```


session {
  session = http::do("GET", "http://www.example.com"); // 该变量会每次请求的时候重新求值
  var2 = 100;
}

````

5. 配置块

和其他所有的Http/Web服务器不同，PL语言支持配置功能。配置块实际上是PL语言的一个特殊代码块，PL的配置块支持所有的动态语法，配置块在代码初始化的时候被执行。用户使用服务器的时候会发现，所有的工作都在PL代码内部完成，服务器的作用只是用于解释执行整个代码仓库罢了。


```

// service配置块
config service {
  // 配置块内可以使用代码语句
  let mydata = http::get("https://www.example.com").body.json();

  // 以下用于配置service配置块内部的属性
  .name = mydata["name"]; // 配置属性必须使用.开头
  ["router"] = mydata["router"]; // 配置属性名称同样可以动态生成，使用[表达式]表示
}

```

6. 迭代器

PL支持携程类型的迭代器，再有的语言中又叫generator。迭代器是带状态，可中断，可重入的特殊函数，类型则为迭代器。用户可以使用PL脚本编写自己觉得合适的迭代器。迭代器必须使用for循环访问。

```

// 定义一个迭代器，迭代器的定义需要使用关键字iter，迭代器内部可以使用yield关键字中断返回。使用yield返回的部分，下次进入迭代器会
// 从该位置恢复执行流程。目前yield必须返回一个Pair对象，否则运行时出错。
iter aIterator() {
  for let i = 0; i < 10; i++ {
    yield (i, "value");
  }
}

// 迭代器可以接受参数，第一次调用的时候传入
iter mapIterator(map) {
  for let k, v = map {
    yield (k, v);
  }
}

// 使用迭代器必须使用 for let i, v = iter iterator 执行
fn useIterator() {
  for let i, v = iter aIterator() {
    println(i, " => ", v);
  }
  
  // 使用带参数的迭代器
  for let i, v = iter mapIterator({"a": 1, "b": 2}) {
    println(i, " => ", v);
  }
}

```


#### 变量生命周期

PL支持4种变量

* 动态变量

动态变量指编译不可见的符号，需要运行时查询宿主环境获得。基本来说，任何在代码单元中不可见的符号都会变成动态变量。


* 全局静态


```

global {
  const1 = http::do("GET", "http://www.example.com"); // will just be initialized once
}

fn foo() {
}

// 这种global区域是无法通过编译的，因为前面已经有个session区域了
global {
...
}

```

* Session变量

每次HTTP请求到来的时候，sesion变量会被重新初始化；请求结束，session变量生命周期结束。

```

session {
  session = http::do("GET", "http://www.example.com"); // will be variable and will be re-evaluated each session
}

fn foo() {
}

// 这种session区域是无法通过编译的，因为前面已经有个session区域了
session {
  ...
}

```

* 局部变量

局部变量使用let或者const表示，在函数或者规则内部定义即可。局部变量拥有最高查询优先级。

```

rule a {
  let a_var = 100;
  a_var = 200;

  const a_const = 10;

  // this is a compile error, since a_const is const local and cannot be modified
  a_const = 20;
}

// same name local variable can be nested inside of different scopes

rule a {
  let xx = 1000;
  {
    let xx = 100;
    {
      let xx = 10;
      assert::eq(xx, 10);
    }
    assert::eq(xx, 100);
  }
  assert::eq(xx, 1000);
}

```


PL中，任何一个符号，编译器会使用如下的查找优先级

1. 局部变量
2. session变量
3. 全局global
4. 动态变量

用户可以通过限定的方式，强制要求某个符号按照某种类型的变量解释

```

rule test {
  // 强制要求a按照session变量查询
  let x = session::a;

  // 强制要求a按照全局变量查询
  let y = global::a;

  // 强制要求a按照动态变量查询
  let z = dynamic::a;
}


```


#### 表达式

* 基本表达式

```
  let a = 1+2*3;
  let b = 1 <= 3+4;
  let c = true != false;
  let d = "abc" + 'def'; // 字符串常量想加
  let e = 2.0 * 3;
  let g = [1, 2, 3, true, null, "Hello World"]; // 数组常量
  let h = {
     'a' : "Hell",
     "b" : " ",
     "c" : "World"
  };  // map常量

  let j = fn(a, b, c) {
    return a+b+c;
  }; // 匿名函数

  // pair常亮表达式
  let pair = ("a", "b");

  // 字表达式
  let subexpression = (1+2);

  // 长字符串，heredoc
  let long_string = ```EOF
	This is a multiple
    line string
  EOF```;

  // 模版字符串，字符串内部使用 {{表达式}} 来进行字符串求值
  // 注意，heredoc不会自动进行字符串模版求值
  let fancy_string = "this is a variable {{e}} and a functino call {{foo(1, 2, 3)}}";

  // 正则表达式常量
  let regex = r"This is a regex string";

  // 正则表达式可以用符号 ~或者!~来进行匹配/不匹配求值
  let matched = "A string" ~ regex; // returns true or false

  // pipe 表达式，用户可以用内置pipe表达式写出流畅的表达式链
  // pipe 表达式实际上是函数调用的语法糖，具体如下：
  // a | b 和函数调用 b(a) 相同
  // a | b(2, 3) 和函数调用 b(a, 2, 3) 相同

  let pipe1 = "string" | str::to_upper;
  let mul10 = fn(a, b) {
    return (a + b) * 10;
  };
  let pipe2 = 1 | mul10(2);

  // PL支持对象方法调用。方法不同于普通函数调用，方法是实现在对象的内部的方法。方法调用使用 ":"触发
  let x = "lower":to_upper(); // string的方法，to_upper

  // 对于map，方法调用和普通的"."调用不一样。

  let obj = {
  };

  // 设置map的成员a为一个闭包函数
  obj.a = fn() {
    return "this is a closure";
  };

  obj.length = fn() {
    return -1;
  };

  // 获得成员a，并且调用该闭包函数
  let aa = obj.a(); // 找到成员a，将a当成函数调用

  // 调用map对象的成员函数length
  let bb = obj:length(); // 返回map的长度，这个例子是2

  // 查找map对象，找到一个叫length的元素，将length当成函数调用
  let cc = obj.length(); // 返回-1
```

* 三元表达式

PL支持三元表达式，与C/C++不同，我们使用类似python的方案，提供更好的可读性。同时，PL支持if表达式，即if整个块语句可以作为表达式使用，if的块的最后一个表达式作为返回值。如果编译器侦测到if的最后一个语句不是表达式，则报错。

```

// 基本的简洁三元表达式

let v1 = "true" if false == false else "false";

// if块作为表达式，注意，if块内部支持如下语言结构
//
// 1) let 定义语句
// 2) const 定义语句
// 3) 循环语句
// 4) 赋值语句
// 5）表达式
//
// 其他语句不支持

let v2 = if false == false {
  foo();
  bar();
  voo();
  100; // 这个100作为if语句块的最终返回值
} else {
  200; // 这个200作为if语句块的最终返回值
};

// 如果if表达式块缺else分支，那么当条件不成立，返回null
let v3 = if false = true {
  100;
};

// 这个例子会导致编译器报错，因为最后一个语句是for循环，for循环不能生成任何值
let v3 = if false == false {
  let x = 100;
  for { ... }; // for loop does not generate any value
};


```

* 高阶函数

PL的函数位高阶（First Order）函数。PL内部的函数包含4种不同类型，如下：

1. 脚本函数
2. 内置函数（Intrinsic Function）
3. Go定义的扩展的Native Function
4. 成员方法

这4种函数的调用方式各不相同

```

fn foo() {
}

// 调用foo函数，定义为脚本的foo函数
let a = foo();

// 假设go运行时导出了bar函数，bar函数为go定义的native function
let _ = bar();

// PL运行时内置了若干特殊的内置函数。内置函数默认有高优先级，用户的定义函数如果和内置函数重名，内置函数
// 的符号被选择
println("Hello World");

// PL同样支持面向对象代码，每个PL对象允许定义成员方法

let v = {};
v:set("a", "b"); // 调用map对象的成员方法set
assert::yes(v::has("a"));

```

由于PL支持高阶函数，因此，每个函数的本身可以被保存为变量。

```

fn foo() {}

let a_foo = foo; // a_foo 变量现在表示foo函数了，用户可以调用a_foo()来调用foo

// 假设运行时暴露动态变量bar为native function
let a_bar = bar; // a_bar表示go定义的native function了

// PL内置的intrinsic函数同样可以被捕获

let a_println = println; // 现在a_println表示println函数了

// 成员函数同样可以被捕获
let a_member = {}:set; // 捕获一个匿名对象map的成员函数set

```

PL的高阶函数能力结合PL的pipe语法，可以很容易写出流畅的面向数据的查询代码


* PL支持表达式异常处理

用户可以通过表达式异常语句限定某个表达式，当某个表达式抛出运行时异常后，代码会恢复到异常处理代码部分接着执行，并且返回正常的表达式。

```

let g = 10;

// 这个例子中g不是函数类型，但是用户使用了表达式异常处理，当g()执行的时候，运行时虚拟机抛出exception，然后虚拟机找到else代码定义的
// 异常处理代码用于恢复执行，最终下面的try语句返回100。
// 注意，try只能处理运行时错误，如果是编译错误，则没有办法。

let xx = try g() else 100;

```

* 模版表达式

模版表达式允许用户直接做模版渲染，并且将结果存储为字符串。模版表达式的优势是模版的编译也是和PL代码便衣一并发生，并不会延后到运行时，后续渲染会复用模版编译结果。


```

// 限定使用go模版，同时模版内部可以使用PL语言的变量
let myGoTemplate = template "go", {"hello" : "world"}, ```EOF
  This is from template {{.hello}} world
EOF```;

```

* 限定符号查询

见前面描述

```

global {
  a = 10;
}

session {
  a = 10;
}

rule xx {

  let a = 10;

  let v1 = session::a;
  let v2 = global::a;
  let v3 = dyanmic::a;
  let v4 = a;
}

```

* 匿名函数和闭包

PL 支持高阶函数，即函数是值。同时支持上值捕获。PL的捕获允许任意嵌套函数的捕获。基本类型按照复制捕获；其他类型则共享内存，只是浅拷贝。上值的binding和局部变量类似，优先级高于session/global/dynamic类型变量。


```

fn nested_func() {
  let a = 1; // a upvalue value

  return fn() {
    return fn() {
      return fn() {
        return a; // capture嵌套上层函数的局部变量为upvalue
      };
    };
  };
}

rule xx {
  assert::eq(nested_func()()()(), 1);
}

// 用户也可以捕获rule内部的上值

rule yy {
  let value = "this is a rule";

  assert::eq(
  	(fn() {
      return value;
    })(), "this is a rule"
  );
}


```


#### 控制流


#### If

和大部分语言类似，PL支持控制流，语法上PL使用elif表示else if。注意，这个地方的if是if语句（statement）而不是表达式。表达式if只在编译器表达式上下文中激活。If语句不需要最后一句话是个表达式。

```

fn foo(flag) {
  if flag == 100 {
    return 100;
  } else {
    return "no";
  }
}

fn bar(flag) {
  if flag == 1 {
    return "A";
  } elif flag == 2 {
    return "B";
  } elif flag == 3 {
    return "C";
  } else {
    return "D";
  }
}

voodoo {
  if request.method == "GET" {
    status => 202;
  } elif request.method == "POST" {
    stauts => 400;
  }
}

```


### 循环

PL只有一个循环关键字，for，但是支持4种不同的循环模式。


1. Forever loop


```

for {
  let x = "I am Crazy!";
}

// 永远不会到达

```


2. 路径循环

允许用户写初始化语句，条件语句和每个迭代结束后的递进语句。


```

for let i = 1; i < 100; i++ {
}

// 如果循环变量在循环体外定义了，那么可以不加关键字let
let j = 0;
for j = 1; j < 100; j++ {
}

// 用户也可以省去某些或者全部的循环条件部分，但是不能省略 ";"

for ; j < 1000; j += 2 {}

for ; ; j+= 3 {}

// 这实际上是个forever loop
for ;; {}

```

3. 迭代器循环

允许用户迭代某些数据结构，比如字符串，list，map，pair或者用户扩展类型（只要用户支持迭代器协议即可）。

```

// 遍历list
for let index, value = [1, 2, 3, 4] {
}

// 便利map
for let key, value = {"a": 1, "b": 2} {
}


// 注意，迭代器循环体必须用let key, val = 格式，否则编译器无法和路径循环区别。下面的for loop会被错误的
// 识别成路径循环，而导致语法错误
for let key = [1, 2, 3, 4] {
}

```

4. 条件循环

常见的C语言while循环，注意条件循环不能用let开始!

```

// 循环到a不等于100

for a != 100 {
}

```


当然，用户可以用常见的循环控制流，比如break和continue。在循环体外使用会造成编译错误。


```

for {
  if xxx {
    break;
  } else {
    continue;
  }
}

```


### 函数调用

代码的任何地方，用户都可以调用其他函数。调用规则需要需要使用emit关键字。


```

fn foo() {
  return 1+2;
}

fn bar() {
  return foo();
}

rule xxx {
  bar();
  // return is not allowed in rule body
  return 10; // error
}

```

### 异常

PL支持异常处理语句。用户可以利用try-else语句去防止某些语句块发生异常，发生异常后，代码会恢复到else部分执行。注意，在某个函数规则调用中出现异常后，VM会持续做stackunwind，直到找到一个else语句块可以处理异常，或者推出虚拟机宣告错误。


```

fn foo() {
  let v;

  try {
    v = dynamic_varaible; // 假设这个变量不存在。
  } else {
    v = 100;
  }

  assert::eq(v, 100);
}

fn not_valid() {
  let v = 10;
  try {
    v = session::a; // session variable a 查找是个编译错误，try无法处理编译错误
  } else {
    v = 100;
  }
}

```

### 模块


PL支持模块编写。每个PL的程序由若干个模块和一个入口代码组成。PL的模块内部不能编写规则，只能编写session，global变量块和定义函数。


#### 模块声明

PL的模块的开头必须要包含模块声明语法，如下

```
// 这个语法定义了该代码表示模块，PL的编译器会自动按照模块的语法解析该代码
module a::cool::module


```

模块声明表示模块的使用命名空间。如上述例子中标明为```a::cool::module```，任何引用该模块的代码调用任何模块内部的符号，都需要加上```a::cool::module```为限定词。

```

// 假设该模块定义的命名空间为a::b
import "a/path/to/module"

let _ = a::b::foo(); // 调用import模块的函数foo需要加上限定词a::b


```

#### 模块内容

PL模块内部只能编写全局变量，session变量和函数。规则不允许在模块中编写。模块内部也可以import其他模块

```
module mymod

// import其他模块是允许的
import "another/module"

global {
  a_global = 1;
  b_global = 2;
}

session {
  a_session = 1;
  b_session = http::do("https://www.tmall.com");
}

fn foo() {}
fn bar() {}

// 这个会造成语法错误，模块内部不允许定义规则
rule xxx {
}

```

### 入口


每个程序都允许有若干模块和一个入口程序。入口程序即没有模块声明的代码（文件）


```

// 注意开头没有module关键字

// 多行import
import (
  "a/b/c/d"
  "a/b/d/c"
  "a/c/b/d"
)

// 可以编写rule
rule {
}

```

### 其他

1. PL会自动侦测循环import，循环import会报错。
2. 其次，递归import不影响模块的限定名字
	1. 比如有个模块a，b import a
	2. c 也import 了a
	3. 入口程序import a，b，c
	4. 实际上a只会被import一次，并且他的模块限定名为其模块内部定义的模块限定名


3. PL的模块实现类似C/C++，不同程序import同一个module会多次编译
	1. PL内部的符号解析尽量使用了静态方案，导致PL必须要知道所有的代码才能编译正确
	2. 除非符号是dynamic，否则PL的符号查询都是数组索引，不涉及map/hash查询
