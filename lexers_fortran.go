package rouge

import (
	"strings"

	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- Fortran ---
// A faithful transcription of Rouge::Lexers::Fortran (rouge 5.0.0,
// lib/rouge/lexers/fortran.rb): Fortran 2008, free form.
//
// The gem's `name`, `kind_param` and `exponent` fragments each carry their own
// /i. Where the rule using them is itself case-insensitive the flag is hoisted
// to a leading (?i); where it is NOT -- the two string rules -- the fragment
// keeps its own (?i:...) scope, because that is what interpolating a
// case-insensitive Regexp into a case-sensitive one does in Ruby.
const (
	fortranName      = `[A-Z][_A-Z0-9]*`
	fortranKindParam = `(?:\d+|(?i:` + fortranName + `))`
	// Inside an already-(?i) rule the fragments need no scope of their own.
	fortranKindParamI = `(?:\d+|` + fortranName + `)`
	fortranExponentI  = `[ED][+-]?\d+`
)

// fortranKeywords is the gem's keyword set. Two-word keywords have their own
// rules further down; Fortran also lets the space between some of them be
// omitted, which is why forms like "endfunction" appear here.
var fortranKeywords = stringSet(`
abstract allocatable allocate assign assignment associate asynchronous
backspace bind block blockdata call case class close codimension
common concurrent contains contiguous continue critical cycle data
deallocate deferred dimension do elemental else elseif elsewhere end
endassociate endblock endblockdata enddo endenum endfile endforall
endfunction endif endinterface endmodule endprogram endselect
endsubmodule endsubroutine endtype endwhere endwhile entry enum
enumerator equivalence exit extends external final flush forall format
function generic goto if implicit import in include inout inquire
intent interface intrinsic is lock module namelist non_overridable
none nopass nullify only open operator optional out parameter pass
pause pointer print private procedure program protected public pure
read recursive result return rewind save select selectcase sequence
stop submodule subroutine target then type unlock use value volatile
wait where while write`)

// fortranTypes: the two-word "double precision" has its own rule, while
// "doubleprecision" is covered here.
var fortranTypes = stringSet(`character complex doubleprecision integer logical real`)

var fortranIntrinsics = stringSet(`
abs achar acos acosh adjustl adjustr aimag aint all allocated anint
any asin asinh associated atan atan2 atanh atomic_define atomic_ref
bessel_j0 bessel_j1 bessel_jn bessel_y0 bessel_y1 bessel_yn bge bgt
bit_size ble blt btest c_associated c_f_pointer c_f_procpointer
c_funloc c_loc c_sizeof ceiling char cmplx command_argument_count
compiler_options compiler_version conjg cos cosh count cpu_time cshift
date_and_time dble digits dim dot_product dprod dshiftl dshiftr
eoshift epsilon erf erfc_scaled erfc execute_command_line exp exponent
extends_type_of findloc floor fraction gamma get_command_argument
get_command get_environment_variable huge hypot iachar iall iand iany
ibclr ibits ibset ichar ieee_class ieee_copy_sign ieee_get_flag
ieee_get_halting_mode ieee_get_rounding_mode ieee_get_status
ieee_get_underflow_mode ieee_is_finite ieee_is_nan ieee_is_normal
ieee_logb ieee_next_after ieee_rem ieee_rint ieee_scalb
ieee_selected_real_kind ieee_set_flag ieee_set_halting_mode
ieee_set_rounding_mode ieee_set_status ieee_set_underflow_mode
ieee_support_datatype ieee_support_denormal ieee_support_divide
ieee_support_flag ieee_support_halting ieee_support_inf
ieee_support_io ieee_support_nan ieee_support_rounding
ieee_support_sqrt ieee_support_standard ieee_support_underflow_control
ieee_unordered ieee_value ieor image_index index int ior iparity
is_contiguous is_iostat_end is_iostat_eor ishft ishftc kind lbound
lcobound leadz len_trim len lge lgt lle llt log_gamma log log10
logical maskl maskr matmul max maxexponent maxloc maxval merge_bits
merge min minexponent minloc minval mod modulo move_alloc mvbits
nearest new_line nint norm2 not null num_images pack parity popcnt
poppar present product radix random_number random_seed range real
repeat reshape rrspacing same_type_as scale scan selected_char_kind
selected_int_kind selected_real_kind set_exponent shape shifta shiftl
shiftr sign sin sinh size spacing spread sqrt storage_size sum
system_clock tan tanh this_image tiny trailz transfer transpose trim
ubound ucobound unpack verify`)

var fortranLexer = func() *RegexLexer {
	b := newRegexLexer("fortran", "Fortran")
	b.filenames("*.f", "*.f90", "*.f95", "*.f03", "*.f08",
		"*.F", "*.F90", "*.F95", "*.F03", "*.F08")

	b.state("root").
		rule(`[\s]+`, TextWhitespace).
		rule(`!.*$`, CommentSingle).
		// ⛔ ^ IS DROPPED FROM THIS RULE AND THE LABEL RULE BELOW, for the
		// reason recorded at length in lexers_cobol.go: the gem scans with
		// StringScanner#skip, where ^ asserts nothing at the scan pointer,
		// while this engine anchors with \G and evaluates ^ against the real
		// buffer. Keeping ^ here would make both rules unreachable off a line
		// start, and the gem reaches them everywhere.
		rule(`#.*$`, CommentPreproc).
		rule(`::|[()\/;,:&\[\]]`, Punctuation).
		// A statement label. The gem notes this does not account for line
		// continuation -- and, because of the ^ above, it also claims any bare
		// integer run: `selected_real_kind(15, 307)` gives Name::Label for both
		// numbers in the gem, which the golden records.
		groupsRule(`(?m)(\s*)([0-9]+)\b`, TextWhitespace, NameLabel).
		// FORMAT statements are strange enough to get their own state.
		groupsRuleT(`(?mi)\b(FORMAT)(\s*)(\()`, []transition{push("format_spec")},
			Keyword, TextWhitespace, Punctuation).
		rule(`(?i)[+-]?(?:(?:\d+[.]\d*|[.]\d+)(?:`+fortranExponentI+`)?`+
			`|\d+(?:`+fortranExponentI+`))`+ // an exponent is mandatory in this branch
			`(?:_`+fortranKindParamI+`)?`, LiteralNumberFloat).
		rule(`(?i)[+-]?\d+(?:_`+fortranKindParamI+`)?`, LiteralNumberInteger).
		rule(`(?i)B'[01]+'|B"[01]+"`, LiteralNumberBin).
		rule(`(?i)O'[0-7]+'|O"[0-7]+"`, LiteralNumberOct).
		rule(`(?i)Z'[0-9A-F]+'|Z"[0-9A-F]+"`, LiteralNumberHex).
		rule(`(?:`+fortranKindParam+`_)?'`, LiteralStringSingle, push("string_single")).
		rule(`(?:`+fortranKindParam+`_)?"`, LiteralStringDouble, push("string_double")).
		rule(`(?i)[.](?:TRUE|FALSE)[.](?:_`+fortranKindParamI+`)?`, KeywordConstant).
		rule(`\*\*|//|==|/=|<=|>=|=>|[-+*/<>=%]`, Operator).
		rule(`(?i)\.(?:EQ|NE|LT|LE|GT|GE|NOT|AND|OR|EQV|NEQV|[A-Z]+)\.`, OperatorWord).
		// Two-word keywords and types. "doubleprecision" is covered by the
		// ordinary keyword rule.
		rule(`(?i)double\s+precision\b`, KeywordType).
		rule(`(?i)go\s+to\b`, Keyword).
		rule(`(?i)sync\s+(?:all|images|memory)\b`, Keyword).
		rule(`(?i)error\s+stop\b`, Keyword).
		cb(`(?mi)`+fortranName, func(l *lexState, m *onigmo.MatchData) {
			match := strings.ToLower(m.Str(0))
			switch {
			case fortranKeywords[match]:
				l.emit(Keyword, m.Str(0))
			case fortranTypes[match]:
				l.emit(KeywordType, m.Str(0))
			case fortranIntrinsics[match]:
				l.emit(NameBuiltin, m.Str(0))
			default:
				l.emit(Name, m.Str(0))
			}
			l.pos = m.End(0)
		})

	b.state("string_single").
		rule(`[^']+`, LiteralStringSingle).
		rule(`''`, LiteralStringEscape).
		rule(`'`, LiteralStringSingle, pop())

	b.state("string_double").
		rule(`[^"]+`, LiteralStringDouble).
		rule(`""`, LiteralStringEscape).
		rule(`"`, LiteralStringDouble, pop())

	b.state("format_spec").
		rule(`'`, LiteralStringSingle, push("string_single")).
		rule(`"`, LiteralStringDouble, push("string_double")).
		rule(`\(`, Punctuation, push("format_spec")).
		rule(`\)`, Punctuation, pop()).
		rule(`,`, Punctuation).
		rule(`[\s]+`, TextWhitespace).
		// Edit descriptors read as a kind of format literal.
		rule(`[^\s'"(),]+`, Literal)

	return b.done()
}()
