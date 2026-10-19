      * a fixed-form comment in the indicator column
       IDENTIFICATION DIVISION.
       PROGRAM-ID. HELLO.
       AUTHOR. A Programmer.

       ENVIRONMENT DIVISION.
       CONFIGURATION SECTION.
       SOURCE-COMPUTER. IBM-370.

       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01  WS-COUNTER      PIC 9(4) VALUE ZERO.
       01  WS-RATE         PIC 9(3)V99 VALUE 12.50.
       01  WS-NAME         PIC X(20) VALUE "A \"quoted\" name".
       01  WS-OTHER        PIC X(10) VALUE 'single'.
       01  WS-FLAG         PIC X VALUE 'Y'.
       01  WS-TOTAL        PIC S9(5) VALUE -42.

       LINKAGE SECTION.
       01  LS-PARM         PIC X(8).

       PROCEDURE DIVISION.
       MAIN-PARA.
           DISPLAY "Hello, world".
           MOVE 1 TO WS-COUNTER.
           COMPUTE WS-TOTAL = WS-COUNTER * WS-RATE + 1.
           IF WS-COUNTER > 0 AND WS-FLAG = 'Y'
               PERFORM VARYING WS-COUNTER FROM 1 BY 1
                   UNTIL WS-COUNTER > 10
                   ADD 1 TO WS-TOTAL
               END-PERFORM
           ELSE
               SUBTRACT 1 FROM WS-TOTAL
           END-IF.
           *> a free-form comment
      * an operator immediately before a word, which the word rule declines
           COMPUTE WS-TOTAL = -ABC.
           EVALUATE TRUE
               WHEN WS-TOTAL < 0
                   DISPLAY "negative"
               WHEN OTHER
                   DISPLAY "non-negative"
           END-EVALUATE.
           STOP RUN.
