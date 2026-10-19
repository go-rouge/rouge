package rouge

import (
	"strings"

	onigmo "github.com/go-ruby-regexp/regexp"
)

// --- COBOL ---
// A faithful transcription of Rouge::Lexers::COBOL (rouge 5.0.0,
// lib/rouge/lexers/cobol.rb).
//
// The gem writes the identifier as /\p{Alpha}[\p{Alnum}-]*/; `[[:alpha:]]` and
// `[[:alnum:]]` are Ruby's other spelling of the same two properties, and are
// what this engine takes.
const cobolID = `[[:alpha:]][[:alnum:]-]*`

var cobolDivisions = stringSet(`IDENTIFICATION ENVIRONMENT DATA PROCEDURE DIVISION`)

var cobolSections = stringSet(`
CONFIGURATION INPUT-OUTPUT FILE WORKING-STORAGE LOCAL-STORAGE LINKAGE SECTION`)

// cobolKeywords is the reserved-word list the gem sources from IBM's COBOL for
// z/OS 6.4 appendix.
var cobolKeywords = stringSet(`
ACCEPT ACCESS ACTIVE-CLASS ADD ADDRESS ADVANCING AFTER ALIGNED ALL ALLOCATE ALPHABET ALPHABETIC ALPHABETIC-LOWER
ALPHABETIC-UPPER ALPHANUMERIC ALPHANUMERIC-EDITED ALSO ALTER ALTERNATE AND ANYCASE ANY APPLY ARE AREA AREAS
ASCENDING ASSIGN AT AUTHOR B-AND B-NOT B-OR B-XOR BASED BASIS BEFORE BEGINNING BINARY BINARY-CHAR BINARY-DOUBLE
BINARY-LONG BINARY-SHORT BIT BLANK BLOCK BOOLEAN BOTTOM BY BYTE-LENGTH CALL CANCEL CBL CD CF CH CHARACTER
CHARACTERS CLASS CLASS-ID CLOCK-UNITS CLOSE COBOL CODE CODE-SET COL COLLATING COLS COLUMN COLUMNS COM-REG COMMA
COMMON COMMUNICATION COMP-1 COMP-2 COMP-3 COMP-4 COMP-5 COMP COMPUTATIONAL-1 COMPUTATIONAL-2
COMPUTATIONAL-3 COMPUTATIONAL-4 COMPUTATIONAL-5 COMPUTATIONAL COMPUTE CONDITION CONSTANT CONTAINS CONTENT
CONTINUE CONTROL CONTROLS CONVERTING COPY CORR CORRESPONDING COUNT CRT CURRENCY CURSOR DATA-POINTER DATE
DATE-COMPILED DATE-WRITTEN DAY DAY-OF-WEEK DBCS DE DEBUG-CONTENTS DEBUG-ITEM DEBUG-LINE DEBUG-NAME DEBUG-SUB-1
DEBUG-SUB-2 DEBUG-SUB-3 DEBUGGING DECIMAL-POINT DECLARATIVES DEFAULT DELETE DELIMITED DELIMITER DEPENDING
DESCENDING DESTINATION DETAIL DISABLE DISPLAY-1 DISPLAY DIVIDE DOWN DUPLICATES DYNAMIC EC EGCS EGI
EJECT ELSE EMI ENABLE END-ACCEPT END-ADD END-CALL END-COMPUTE END-DELETE END-DISPLAY END-DIVIDE END-EVALUATE
END-EXEC END-IF END-INVOKE END-JSON END-MULTIPLY END-OF-PAGE END-PERFORM END-READ END-RECEIVE END-RETURN
END-REWRITE END-SEARCH END-START END-STRING END-SUBTRACT END-UNSTRING END-WRITE END-XML ENDING END ENTER ENTRY
EO EOP EQUAL ERROR ESI EVALUATE EVERY EXCEPTION EXCEPTION-OBJECT EXEC EXECUTE EXIT EXTEND EXTERNAL
FACTORY FALSE FD FILE-CONTROL FILLER FINAL FIRST FLOAT-EXTENDED FLOAT-LONG FLOAT-SHORT FOOTING FOR FORMAT
FREE FROM FUNCTION FUNCTION-ID FUNCTION-POINTER GENERATE GET GIVING GLOBAL GO GOBACK GREATER GROUP GROUP-USAGE
HEADING HIGH-VALUE HIGH-VALUES I-O-CONTROL I-O ID IF IN INDEX INDEXED INDICATE INHERITS INITIAL
INITIALIZE INITIATE INPUT INSERT INSPECT INSTALLATION INTERFACE INTERFACE-ID INTO INVALID INVOKE
IS JAVA JNIENVPTR JSON JSON-CODE JSON-STATUS JUST JUSTIFIED KANJI KEY LABEL LAST LEADING LEFT LENGTH LESS LIMIT
LIMITS LINAGE-COUNTER LINAGE LINE-COUNTER LINES LINE LOCALE LOCK LOW-VALUE LOW-VALUES
MEMORY MERGE MESSAGE METHOD METHOD-ID MINUS MODE MODULES MORE-LABELS MOVE MULTIPLE MULTIPLY NATIONAL
NATIONAL-EDITED NATIVE NEGATIVE NESTED NEXT NO NOT NULL NULLS NUMBER NUMERIC NUMERIC-EDITED OBJECT
OBJECT-COMPUTER OBJECT-REFERENCE OCCURS OF OFF OMITTED ON OPEN OPTIONAL OPTIONS OR ORDER ORGANIZATION
OTHER OUTPUT OVERFLOW OVERRIDE PACKED-DECIMAL PADDING PAGE PAGE-COUNTER PASSWORD PERFORM PF PH PIC PICTURE
PLUS POINTER- POINTER-31 POINTER-32 POINTER-64 POINTER POSITION POSITIVE PRESENT PRINTING
PROCEDURE-POINTER PROCEDURES PROCEED PROCESSING PROGRAM-ID PROGRAM-POINTER PROGRAM PROPERTY PROTOTYPE
PURGE QUEUE QUOTE QUOTES RAISE RAISING RANDOM RD READ READY RECEIVE RECORD RECORDING RECORDS RECURSIVE REDEFINES
REEL REFERENCE REFERENCES RELATIVE RELEASE RELOAD REMAINDER REMOVAL RENAMES REPLACE REPLACING REPORT REPORTING
REPORTS REPOSITORY RERUN RESERVE RESET RESUME RETRY RETURN RETURN-CODE RETURNING REVERSED REWIND REWRITE RF RH
RIGHT ROUNDED RUN SAME SCREEN SD SEARCH SECTION SECURITY SEGMENT SEGMENT-LIMIT SELECT SELF SEND SENTENCE
SEPARATE SEQUENCE SEQUENTIAL SERVICE SET SHARING SHIFT-IN SHIFT-OUT SIGN SIZE SKIP1 SKIP2 SKIP3
SORT-CONTROL SORT-CORE-SIZE SORT-FILE-SIZE SORT-MERGE SORT-MESSAGE SORT-MODE-SIZE SORT-RETURN SORT
SOURCE-COMPUTER SOURCES SOURCE SPACE SPACES SPECIAL-NAMES SQL SQLIMS STANDARD-1 STANDARD-2 STANDARD START STATUS STOP
STRING SUB-QUEUE-1 SUB-QUEUE-2 SUB-QUEUE-3 SUBTRACT SUM SUPER SUPPRESS SYMBOLIC SYNC SYNCHRONIZED SYSTEM-DEFAULT
TABLE TALLY TALLYING TAPE TERMINAL TERMINATE TEST TEXT THAN THEN THROUGH THRU TIME TIMES TITLE TO TOP TRACE
TRAILING TRUE TYPE TYPEDEF UNIT UNIVERSAL UNLOCK UNSTRING UNTIL UP UPON USAGE USE USER-DEFAULT USING UTF-8
VAL-STATUS VALID VALIDATE VALIDATE-STATUS VALUE VALUES VARYING VOLATILE WHEN WHEN-COMPILED WITH WORDS
WRITE WRITE-ONLY XML-CODE XML-EVENT XML-INFORMATION XML-NAMESPACE XML-NAMESPACE-PREFIX
XML-NNAMESPACE XML-NNAMESPACE-PREFIX XML-NTEXT XML-SCHEMA XML-TEXT XML ZERO ZEROES ZEROS`)

var cobolLexer = func() *RegexLexer {
	b := newRegexLexer("cobol", "COBOL")
	b.filenames("*.cob", "*.cbl", "*.cpy", "*.cpb")

	b.state("root").
		// Comments first: a fixed-form indicator column, or a Debug line.
		//
		// ⛔ THE GEM'S ^ DOES NOT MEAN "LINE START" HERE, and transcribing it
		// literally makes this rule unreachable. Rouge scans with
		// StringScanner#skip, and measured against rouge 5.0.0 the rule fires
		// wherever the scan pointer is:
		//
		//	"X      * c"               -> Name "X", then Comment::Special "      * c"
		//	11 spaces then "*> c"      -> 5 Text::Whitespace, then Comment::Special
		//	                              from the sixth space
		//
		// so the six spaces are counted FROM THE POINTER, not from the column.
		// This engine anchors every rule with \G and evaluates ^ against the
		// real buffer, where neither of those positions is a line start. \G
		// alone already says "at the scan position", which is exactly what the
		// gem's ^ amounts to, so it is dropped. Documented in the README.
		rule(`(?:      \*).*|(?:Debug \*).*`, CommentSpecial).
		// Strings
		rule(`"`, LiteralStringDouble, push("string_double")).
		rule(`'`, LiteralStringSingle, push("string_single")).
		// Keywords, divisions and sections, matched case-insensitively and
		// compared upper-cased, as the gem does.
		//
		// ⛔ THE GEM'S (?<![\w-]) GUARD IS DROPPED, for the same reason as the
		// ^ above: it asserts nothing where the gem evaluates it. This engine
		// refuses it outright as a variable-width lookbehind, and the obvious
		// replacement -- declining the match when the preceding byte is a word
		// character or a dash -- is WRONG, measured against the gem:
		//
		//	COMPUTE WS-TOTAL = -ABC.
		//
		// gives Operator "-" and then Name "ABC", Punctuation "." in rouge
		// 5.0.0. A preceding-byte check declines there and leaves "ABC." to
		// the catch-all, which is a different token stream. The trailing
		// (?![\w-]) is an ordinary lookahead and stays. Documented in the
		// README.
		cb(`(?i)`+cobolID+`(?![\w-])`, func(l *lexState, m *onigmo.MatchData) {
			word := strings.ToUpper(m.Str(0))
			switch {
			case cobolDivisions[word]:
				l.emit(KeywordDeclaration, m.Str(0))
			case cobolSections[word]:
				l.emit(KeywordNamespace, m.Str(0))
			case cobolKeywords[word]:
				l.emit(Keyword, m.Str(0))
			default:
				l.emit(Name, m.Str(0))
			}
			l.pos = m.End(0)
		}).
		// Numbers
		rule(`[-+]?\b\d+(?:\.\d+)?\b`, LiteralNumber).
		// Punctuation
		rule(`[.,;:()]`, Punctuation).
		// Free-form comments
		rule(`\*>.*`, CommentSingle).
		// Operators
		rule(`[+\-*\/><=]`, Operator).
		// Whitespace remaining
		rule(`\s`, TextWhitespace).
		// Anything else remaining
		rule(`.+`, Text)

	// The gem carries three TODOs here, kept because they describe what this
	// lexer does NOT do: it does not model COBOL escaping, it lets a string
	// opened with " be closed by ', and it refuses multi-line strings.
	b.state("string_double").
		rule(`[^"\\\n]+`, LiteralString).
		rule(`\\.`, LiteralStringEscape).
		rule(`"`, LiteralStringDouble, pop()).
		rule(`\n`, Error) // a string running to the next line is an error

	b.state("string_single").
		rule(`[^'\\\n]+`, LiteralString).
		rule(`\\.`, LiteralStringEscape).
		rule(`'`, LiteralStringSingle, pop()).
		rule(`\n`, Error)

	return b.done()
}()
