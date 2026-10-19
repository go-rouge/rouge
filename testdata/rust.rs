//! An inner doc comment.
//// four slashes is an ordinary comment again
/// An outer doc comment.
// a plain line comment
/**/
/*** a plain block /* nested */ still plain ***/
/** a doc block /* nested plain is still doc */ */

#![allow(dead_code)]
#[derive(Debug, Clone, PartialEq)]
pub struct Buffer<T> {
    items: Vec<T>,
    len: usize,
}

const MAX: u32 = 1_000;
static GREETING: &str = "hello \"world\"\n";

macro_rules! twice {
    ($x:expr) => {
        $x + $x
    };
}

impl<T: Clone> Buffer<T> {
    pub fn new() -> Self {
        Buffer { items: Vec::new(), len: 0 }
    }

    pub async fn push(&mut self, item: T) -> Result<(), String> {
        self.items.push(item);
        self.len += 1;
        Ok(())
    }

    fn numbers(&self) {
        // non-ASCII identifiers, so the XID substitution is measured
        let café = 1;
        let Ωmega = 2;
        let _ = (café, Ωmega);
        let a = 42;
        let b = 42u8;
        let c = 0xdead_beef;
        let d = 0b1010_1010i64;
        let e = 0o755;
        let f = 1_000.5;
        let g = 1.5e-3f64;
        let h = 2.;
        let _ = (a, b, c, d, e, f, g, h);
    }

    fn strings(&self) {
        let s = "escaped \t \u{1F600} \x41";
        let r = r#"raw "quotes" and \no escapes"#;
        let b = b"bytes";
        let ch = 'x';
        let nl = '\n';
        let _ = (s, r, b, ch, nl);
    }
}

pub trait Shape {
    fn area(&self) -> f64;
}

fn main() {
    let mut buf = Buffer::<i32>::new();
    println!("{:?} {}", buf, twice!(3));
    let v = vec![1, 2, 3];
    for x in v.iter() {
        match x {
            1 => println!("one"),
            _ => (),
        }
    }
    let opt: Option<&i32> = v.first();
    if let Some(n) = opt {
        let _ = n.to_string();
    }
    let lt: &'static str = GREETING;
    let _ = (lt, MAX, buf.len);
}
