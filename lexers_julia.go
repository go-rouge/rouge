// Copyright (c) the go-rouge/rouge authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rouge

import (
	"fmt"
	"strings"
	"unicode"
)

// juliaWordSets holds the word sets the prefilters consult, so a test can feed
// the lexer every one of their words.
var juliaWordSets []map[string]bool

// wordSet lists the words of one of the /x word-set patterns above: whitespace
// is dropped (the gem's /x ignores it, so `mutable struct` is `mutablestruct`),
// the `(?x:(?:` ... `)\b)` wrapper is peeled off, and `(LOAD|DEPOT)_PATH` is
// spelled out. Every word is ASCII word bytes or, for π and ℯ, a single
// non-ASCII letter that wordIn never consults.
func wordSet(pattern string) map[string]bool {
	p := strings.Join(strings.Fields(pattern), "")
	p = strings.TrimSuffix(strings.TrimPrefix(p, `(?x:(?:`), `)\b)`)
	p = strings.Replace(p, `(LOAD|DEPOT)_PATH`, `LOAD_PATH|DEPOT_PATH`, 1)
	set := map[string]bool{}
	for _, w := range strings.Split(p, "|") {
		set[w] = true
	}
	return set
}

// classOf spells the members of a Unicode range table as the body of a
// character class: ASCII members as \xHH, the rest as themselves. It stands in
// for a \p{...} property name the engine does not provide.
func classOf(t *unicode.RangeTable) string {
	var b strings.Builder
	emit := func(lo, hi, stride rune) {
		for r := lo; r <= hi; r += stride {
			if r < 0x80 {
				fmt.Fprintf(&b, `\x%02X`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	for _, r := range t.R16 {
		emit(rune(r.Lo), rune(r.Hi), rune(r.Stride))
	}
	for _, r := range t.R32 {
		emit(rune(r.Lo), rune(r.Hi), rune(r.Stride))
	}
	return b.String()
}

var shebangLine = compileRE(`\A\s*#!(.*)$`)

// shebangDetector is Rouge::TextAnalyzer#shebang?: the text starts with a `#!`
// line and that line contains name as a whole word followed by whitespace or
// the end of the line (`#!/usr/bin/env julia`, `#!/bin/julia -O3`).
func shebangDetector(name string) func(string) bool {
	word := compileRE(`.*?\b` + name + `(?:\s|$)`)
	return func(text string) bool {
		m := shebangLine.MatchAt(text, 0)
		if m == nil {
			return false
		}
		return word.MatchString(m.Str(1))
	}
}

// --- Julia ---
// A transcription of Rouge::Lexers::Julia (lib/rouge/lexers/julia.rb, rouge
// v5.1.0). The gem's mimetypes ('text/x-julia', 'application/x-julia') have no
// counterpart in this library's lexer metadata.
//
// Three deviations from the text of julia.rb, each measured against the gem
// (see the README, "Documented simplifications"):
//
//   - the trailing-transpose rule `(?<=[.\w)\]])'+` is omitted: Rouge scans with
//     StringScanner#skip, where a positive lookbehind never sees the byte before
//     the scan pointer, so the rule can never fire in the gem;
//   - the leading `\b` of BUILTINS, KEYWORDS and TYPES is omitted: every
//     alternative starts with a word character, and at the scan pointer the gem
//     evaluates `\b` as at the start of the string, i.e. always true;
//   - detect? uses the shebang check only, like the other detectors here.

var juliaLexer = func() *RegexLexer {
	// The gem's three word sets and OPERATORS are /x regexps: whitespace and line
	// breaks inside them are ignored, which is why `mutable struct` in KEYWORDS
	// is really `mutablestruct` and can never match a space-separated pair. The
	// (?x: ... ) wrapper reproduces that exactly, with the alternatives pasted
	// from the gem unchanged.
	const builtins = `(?x:(?:
                              true      | false    | missing | nothing
                            | Inf       | Inf16    | Inf32   | Inf64
                            | NaN       | NaN16    | NaN32   | NaN64
                            | stdout    | stderr   | stdin   | devnull
                            | pi        | π        | ℯ       | im
                            | ARGS      | C_NULL   | ENV     | ENDIAN_BOM
                            | VERSION   | undef    | (LOAD|DEPOT)_PATH
                            )\b)`

	const keywords = `(?x:(?:
                              function | return | module | import | export
                            | if       | else   | elseif | end    | for
                            | in       | isa    | while  | try    | catch
                            | const    | local  | global | using  | struct
                            | mutable struct    | abstract type   | finally
                            | begin    | do     | quote  | macro  | for outer
                            | where
                            )\b)`

	const types = `(?x:(?:
                              ARPACKException | AbstractArray | AbstractChannel
                            | AbstractChar | AbstractDict | AbstractDisplay
                            | AbstractFloat | AbstractIrrational | AbstractLogger
                            | AbstractMatrix | AbstractREPL | AbstractRNG
                            | AbstractRange | AbstractSerializer | AbstractSet
                            | AbstractSparseArray | AbstractSparseMatrix | AbstractSparseVector
                            | AbstractString | AbstractUnitRange | AbstractVecOrMat
                            | AbstractVector | AbstractWorkerPool | Adjoint
                            | Any | ArgumentError | Array
                            | AssertionError | Base64DecodePipe | Base64EncodePipe
                            | BasicREPL | Bidiagonal | BigFloat
                            | BigInt | BitArray | BitMatrix
                            | BitSet | BitVector | Bool
                            | BoundsError | BunchKaufman | CachingPool
                            | CapturedException | CartesianIndex | CartesianIndices
                            | Cchar | Cdouble | Cfloat
                            | Channel | Char | Cholesky
                            | CholeskyPivoted | Cint | Cintmax_t
                            | Clong | Clonglong | ClusterManager
                            | Cmd | Colon | Complex
                            | ComplexF16 | ComplexF32 | ComplexF64
                            | CompositeException | Condition | ConsoleLogger
                            | Cptrdiff_t | Cshort | Csize_t
                            | Cssize_t | Cstring | Cuchar
                            | Cuint | Cuintmax_t | Culong
                            | Culonglong | Cushort | Cvoid
                            | Cwchar_t | Cwstring | DataType
                            | Date | DateFormat | DatePeriod
                            | DateTime | Day | DenseArray
                            | DenseMatrix | DenseVecOrMat | DenseVector
                            | Diagonal | Dict | DimensionMismatch
                            | Dims | DivideError | DomainError
                            | EOFError | Eigen | Enum
                            | ErrorException | Exception | ExponentialBackOff
                            | Expr | FDWatcher | Factorization
                            | FileMonitor | Float16 | Float32
                            | Float64 | FolderMonitor | Function
                            | GeneralizedEigen | GeneralizedSVD | GeneralizedSchur
                            | GenericArray | GenericDict | GenericSet
                            | GenericString | GitConfig | GitRepo
                            | GlobalRef | HMAC_CTX | HTML
                            | Hermitian | Hessenberg | Hour
                            | IO | IOBuffer | IOContext
                            | IOStream | IPAddr | IPv4
                            | IPv6 | IdDict | IndexCartesian
                            | IndexLinear | IndexStyle | InexactError
                            | InitError | Int | Int128
                            | Int16 | Int32 | Int64
                            | Int8 | Integer | InterruptException
                            | InvalidStateException | Irrational | KeyError
                            | LAPACKException | LDLt | LQ
                            | LU | LinRange | LineEditREPL
                            | LineNumberNode | LinearIndices | LoadError
                            | LogLevel | LowerTriangular | MIME
                            | Matrix | MersenneTwister | Method
                            | MethodError | Microsecond | Millisecond
                            | Minute | Missing | MissingException
                            | Module | Month | NTuple
                            | NamedTuple | Nanosecond | Nothing
                            | NullLogger | Number | OrdinalRange
                            | OutOfMemoryError | OverflowError | PackageMode
                            | PackageSpec | Pair | PartialQuickSort
                            | Period | PermutedDimsArray | Pipe
                            | PollingFileWatcher | PosDefException | ProcessExitedException
                            | Ptr | QR | QRPivoted
                            | QuoteNode | RandomDevice | RankDeficientException
                            | Rational | RawFD | ReadOnlyMemoryError
                            | Real | ReentrantLock | Ref
                            | Regex | RegexMatch | RemoteChannel
                            | RemoteException | RoundingMode | SHA1_CTX
                            | SHA224_CTX | SHA256_CTX | SHA2_224_CTX
                            | SHA2_256_CTX | SHA2_384_CTX | SHA2_512_CTX
                            | SHA384_CTX | SHA3_224_CTX | SHA3_256_CTX
                            | SHA3_384_CTX | SHA3_512_CTX | SHA512_CTX
                            | SVD | Schur | Second
                            | SegmentationFault | Serializer | Set
                            | SharedArray | SharedMatrix | SharedVector
                            | Signed | SimpleLogger | SingularException
                            | Some | SparseMatrixCSC | SparseVector
                            | StackOverflowError | StepRange | StepRangeLen
                            | StreamREPL | StridedArray | StridedMatrix
                            | StridedVecOrMat | StridedVector | String
                            | StringIndexError | SubArray | SubString
                            | SubstitutionString | SymTridiagonal | Symbol
                            | Symmetric | SystemError | TCPSocket
                            | Task | TestSetException | Text
                            | TextDisplay | Time | TimePeriod
                            | TimeType | TimeZone | Timer
                            | Transpose | Tridiagonal | Tuple
                            | Type | TypeError | TypeVar
                            | UDPSocket | UInt | UInt128
                            | UInt16 | UInt32 | UInt64
                            | UInt8 | UTC | UUID
                            | UndefInitializer | UndefKeywordError | UndefRefError
                            | UndefVarError | UniformScaling | Union
                            | UnionAll | UnitLowerTriangular | UnitRange
                            | UnitUpperTriangular | Unsigned | UpgradeLevel
                            | UpperTriangular | Val | Vararg
                            | VecElement | VecOrMat | Vector
                            | VersionNumber | WeakKeyDict | WeakRef
                            | Week | WorkerConfig | WorkerPool
                            | Year
                            )\b)`

	const operators = `(?x: \+      | =        | -     | \*   | \/
                              | \\    | &        | \|    | \$   | ~
                              | \^    | %        | !     | >>>  | >>
                              | <<    | &&       | \|\|  | \+=  | -=
                              | \*=   | \/=      | \\=   | ÷=   | %=
                              | \^=   | &=       | \|=   | \$=  | >>>=
                              | >>=   | <<=      | ==    | !=   | ≠
                              | <=    | ≤        | >=    | ≥    | \.
                              | ::    | <:       | ->    | \?   | \.\*
                              | \.\^  | \.\\     | \.\/  | \\   | <
                              | >     | ÷        | >:    | :    | ===
                              | !==   | =>
                            )`

	const punctuation = `[\[\]{}\(\),;]`
	// The gem's NAME_RE is `[\p{L}\p{Nl}\p{S}_][\p{Word}\p{S}\p{Po}!]*`. The
	// engine has no \p{Po} (Other_Punctuation), so it is spelled out from Go's
	// unicode.Po table, one code point per entry. \p{P} would NOT do: it also
	// holds the dashes and brackets, which would make `f(x)` one name.
	nameRE := `[\p{L}\p{Nl}\p{S}_][\p{Word}\p{S}` + classOf(unicode.Po) + `!]*`

	typeWords, keywordWords, builtinWords := wordSet(types), wordSet(keywords), wordSet(builtins)
	juliaWordSets = []map[string]bool{typeWords, keywordWords, builtinWords}

	b := newRegexLexer("julia", "Julia", "jl")
	b.filenames("*.jl")
	b.detectWith(shebangDetector("julia"))

	b.state("root").
		rule(`\n`, Text).
		// The gem writes `[^\S\n]+`. The engine evaluates a negated class that
		// holds \S byte-wise, so it swallows every non-ASCII character (`2 * π`
		// came out as Text " π "); `(?:(?!\n)\s)+` is the same set -- whitespace
		// except newline -- spelled without the double negation.
		rule(`(?:(?!\n)\s)+`, Text).
		rule(`#=`, CommentMultiline, push("blockcomment")).
		rule(`#.*$`, Comment).
		rule(operators, Operator).when(startsWithAny(`+=-*/\&|$~^%!><.:?`)).
		rule(`\\\n`, Text).
		rule(`\\`, Text).
		// functions and macros
		groupsRuleT(`(function|macro)((?:\s|\\\s)+)`, []transition{push("funcname")}, Keyword, NameFunction).when(startsWithAny("fm")).
		// types
		groupsRuleT(`((?:mutable )?struct|(?:abstract|primitive) type)((?:\s|\\\s)+)`,
			[]transition{push("typename")}, Keyword, NameClass).when(startsWithAny("msap")).
		rule(types, KeywordType).when(wordIn(typeWords)).
		// keywords
		rule(`(local|global|const)\b`, KeywordDeclaration).when(startsWithAny("lgc")).
		rule(keywords, Keyword).when(wordIn(keywordWords)).
		rule(builtins, NameBuiltin).when(wordIn(builtinWords)).
		// backticks
		rule("`.*?`", LiteralStringBacktick).when(startsWithAny("`")).
		// chars
		rule(`'(\\.|\\[0-7]{1,3}|\\x[a-fA-F0-9]{1,3}|\\u[a-fA-F0-9]{1,4}|\\U[a-fA-F0-9]{1,6}|[^\\\'\n])'`, LiteralStringChar).when(startsWithAny("'")).
		// strings
		rule(`(?:[IL])"`, LiteralString, push("string")).when(startsWithAny("IL")).
		rule(`[E]?"`, LiteralString, push("string")).when(startsWithAny(`E"`)).
		// names
		rule(`@[\w.]+`, NameDecorator).when(startsWithAny("@")).
		rule(`(?:[a-zA-Z_¡-￿]|[က-ჿ])(?:[a-zA-Z_0-9¡-￿]|[က-ჿ])*!*`, Name).
		rule(punctuation, Other).
		// numbers
		rule(`(\d+(_\d+)+\.\d*|\d*\.\d+(_\d+)+)([eEf][+-]?[0-9]+)?`, LiteralNumberFloat).
		rule(`(\d+\.\d*|\d*\.\d+)([eEf][+-]?[0-9]+)?`, LiteralNumberFloat).
		rule(`\d+(_\d+)+[eEf][+-]?[0-9]+`, LiteralNumberFloat).
		rule(`\d+[eEf][+-]?[0-9]+`, LiteralNumberFloat).
		rule(`0b[01]+(_[01]+)+`, LiteralNumberBin).
		rule(`0b[01]+`, LiteralNumberBin).
		rule(`0o[0-7]+(_[0-7]+)+`, LiteralNumberOct).
		rule(`0o[0-7]+`, LiteralNumberOct).
		rule(`0x[a-fA-F0-9]+(_[a-fA-F0-9]+)+`, LiteralNumberHex).
		rule(`0x[a-fA-F0-9]+`, LiteralNumberHex).
		rule(`\d+(_\d+)+`, LiteralNumberInteger).
		rule(`\d+`, LiteralNumberInteger)

	b.state("funcname").
		rule(nameRE, NameFunction, pop()).
		rule(`\([^\s\w{]{1,2}\)`, Operator, pop()).
		rule(`[^\s\w{]{1,2}`, Operator, pop())

	b.state("typename").
		rule(nameRE, NameClass, pop())

	b.state("stringescape").
		rule(`\\([\\abfnrtv"\']|\n|N\{.*?\}|u[a-fA-F0-9]{4}|U[a-fA-F0-9]{8}|x[a-fA-F0-9]{2}|[0-7]{1,3})`,
			LiteralStringEscape)

	b.state("blockcomment").
		rule(`[^=#]`, CommentMultiline).
		rule(`#=`, CommentMultiline, push("blockcomment")).
		rule(`\=#`, CommentMultiline, pop()).
		rule(`[=#]`, CommentMultiline)

	b.state("string").
		mixin("stringescape").
		rule(`"`, LiteralString, pop()).
		rule(`\\\\|\\"|\\\n`, LiteralStringEscape). // included here for raw strings
		rule(`\$(\(\w+\))?[-#0 +]*([0-9]+|[*])?(\.([0-9]+|[*]))?`, LiteralStringInterpol).
		rule(`[^\\"$]+`, LiteralString).
		// quotes, dollar signs, and backslashes must be parsed one at a time
		rule(`["\\]`, LiteralString).
		// unhandled string formatting sign
		rule(`\$`, LiteralString)

	return b.done()
}()
