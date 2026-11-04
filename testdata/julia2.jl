#!/usr/bin/env julia
# Edge cases the visual sample does not reach.
A = [1 2; 3 4]
B = A'            # trailing transpose: the gem's lookbehind never fires
C = A'' + x.'y
2if x end
3in y
mutable struct Point{T<:Real}
    x::T
end
primitive type Octet 8 end
abstract type Shape end
struct (Pair)
end
function (+)(a, b) a end
function ∘(f, g) f end
macro m(x) x end
local a = 1; global b = 2; const c = 3
for outer i in 1:3 end
@time @inbounds x[1]
`ls -l`
'a' '\n' '\x41' 'é' '\U0001F600' '\101' 'ab'
"plain \"quoted\" \\ \$ é \x41 \101 \N{DIGIT ONE} \
continued"
I"raw\n" L"wide" E"esc"
"printf %d %5.2f %-3s %(name)s $x $(y) $"
#= block #= nested =# still =# after
#= unterminated block
π ℯ im Inf32 NaN16 LOAD_PATH DEPOT_PATH
x ≠ y ≤ z ≥ w ÷ 2 ÷= 3 .* .^ .\ ./ <: >: === !== => -> :: ?
αβγ = 1_000.5e3 + 0b1_01 + 0o7_7 + 0xdead_beef + 1_0 + .5 + 5. + 1e5 + 1f3 + 1_0e2
Ωmega!!! = Vector{Int64}(undef, 3)
\
x \
y
[a, b]{c}(d);
x = π + y ∈ z 语言 ∘ w
function f′(x) x end
function a·b(x) x end
struct S′ end
function (-)(x) x end
function [](x) x end
