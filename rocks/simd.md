# SIMD Reference

Each `simd.Uint32s` operation runs independently on every `uint32` lane.

```text
a = [a0 a1 a2 a3]
b = [b0 b1 b2 b3]

a.And(b) = [a0 & b0, a1 & b1, a2 & b2, a3 & b3]
```

## Bitwise Operations

### AND

`a.And(b)` is equivalent to `a & b`.

```text
  11001010
& 10101100
= 10001000
```

Keeps bits that are 1 in both inputs. Commonly used to isolate a field.

### OR

`a.Or(b)` is equivalent to `a | b`.

```text
  11000000
| 00101010
= 11101010
```

Keeps bits that are 1 in either input. Used to combine fields that occupy different bit ranges. OR is not addition:

```text
0101 | 0011 = 0111
0101 + 0011 = 1000
```

### XOR

`a.Xor(b)` is equivalent to `a ^ b`.

```text
  11001010
^ 10101100
= 01100110
```

Keeps bits that differ and clears bits that are equal.

### AND NOT

`a.AndNot(b)` is equivalent to `a &^ b`.

```text
   11001010
&^ 00101100
 = 11000010
```

Keeps bits from `a` except where `b` contains a 1. The bits in `b` select which bits to clear.

### NOT

`a.Not()` is equivalent to `^a`.

```text
^ 11001010
= 00110101
```

Flips every 0 to 1 and every 1 to 0.

## Bit Movement

### Shift Left

`a.ShiftAllLeft(2)` is equivalent to `a << 2`.

```text
  00101101 << 2
= 10110100
```

Moves bits left, drops bits past the left edge, and adds zeros on the right.

### Shift Right

`a.ShiftAllRight(2)` is equivalent to `a >> 2`.

```text
  10110100 >> 2
= 00101101
```

Moves bits right, drops bits past the right edge, and adds zeros on the left.

### Rotate Left

`a.RotateAllLeft(2)` is equivalent to `bits.RotateLeft32(a, 2)`.

```text
  10110001 rotate left 2
= 11000110
```

Moves bits left and wraps the shifted-out bits around to the right edge.

### Rotate Right

`a.RotateAllRight(2)` is equivalent to `bits.RotateLeft32(a, -2)`.

```text
  10110001 rotate right 2
= 01101100
```

Moves bits right and wraps the shifted-out bits around to the left edge.

## Arithmetic

### Add

```text
  00000101  (5)
+ 00000011  (3)
= 00001000  (8)
```

### Subtract

```text
  00000101  (5)
- 00000011  (3)
= 00000010  (2)
```

### Multiply

```text
  00000101  (5)
* 00000011  (3)
= 00001111  (15)
```

Each operation performs its arithmetic independently in every lane.

## Lane Comparisons And Selection

### Compare

```text
currentValues  = [0011  0101  0111  1001]
expectedValues = [0011  0010  0111  0001]
matchingLanes  = [true  false true  false]
```

```go
matchingLanes := currentValues.Equal(expectedValues)
```

This produces one boolean per lane, not integer zero or one. `Less`, `LessEqual`, `Greater`, `GreaterEqual`, and `NotEqual` produce masks in the same way.

### Select

```text
updatedValues  = [0001  0010  0011  0100]
originalValues = [1001  1010  1011  1100]
updateLanes    = [true  false true  false]
selectedValues = [0001  1010  0011  1100]
```

```go
selectedValues := updatedValues.IfElse(
    updateLanes,
    originalValues,
)
```

SIMD examines each lane independently. A true update lane copies from `updatedValues`; a false update lane copies from `originalValues`.

### Clear Disabled Lanes

```text
inputValues    = [0001  0010  0011  0100]
enabledLanes   = [true  false true  false]
filteredValues = [0001  0000  0011  0000]
```

```go
filteredValues := inputValues.Masked(enabledLanes)
```

A true enabled lane retains its complete input value. A false enabled lane replaces its complete input value with zero.
