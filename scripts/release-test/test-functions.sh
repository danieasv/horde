#!/bin/bash
#
# Test functions. Call "start_test", "end_test", "section" and "step" while doing tests
# any non-zero returns fails the test.
# Tests are numbered incrementally and consists of sections and steps within
# each section. Tests either pass or they don't. The tests runs with the credo
# "failure is not an option" -- if one of the steps fails, everything fails.
#

TEST_COUNT=0
SECTION_COUNT=0
STEP_COUNT=0
CURRENT_TEST="none"
STEP_MSG=""
TESTSPEC=release/archives/test-spec-${VERSION}.tex
IN_STEP=0

function begin_run {
    if [ -f ${TESTSPEC} ]; then
        rm ${TESTSPEC}
    fi
    DATE=$(date +"%Y-%m-%d %H:%M")
    NAME=$(reto hashname)
    COMMIT=$(reto hash)
    cat >> ${TESTSPEC} <<EOL
\documentclass{article}
\usepackage[utf8]{inputenc}
\usepackage{booktabs}
\usepackage{array}
\usepackage{float}
\usepackage{longtable}
\usepackage[a4paper, total={6in, 9in}]{geometry}
\title{%
    Horde release test \\\\
    \large Version: $TEST_VERSION ($NAME) \\\\
     Commit hash: $COMMIT}
\author{ee@telenordigital.com}
\date{$DATE}

\begin{document}
\newcolumntype{A}{>{\raggedright\arraybackslash}p{1cm}}
\newcolumntype{B}{>{\raggedright\arraybackslash}p{5cm}}
\newcolumntype{C}{>{\raggedright\arraybackslash}p{8cm}}
\renewcommand\arraystretch{1.5}

\maketitle
% begin_run
EOL
}

function end_run {
    end_table_if_step
    IN_STEP=0

    echo "% end_run" >> ${TESTSPEC}
    echo "\end{document}" >> ${TESTSPEC}

    echo "Test spec is in ${TESTSPEC}"
}

# Start the test block. This resets the section counters and increments the
# test counter.
TEST_VERSION="unknown"

function start_test {
    end_table_if_step
    IN_STEP=0
    CURRENT_TEST=$1
    SECTION_COUNT=0
    let TEST_COUNT=TEST_COUNT+1
    echo "#####################################################################"
    echo "# Start test  ${TEST_COUNT}.${CURRENT_TEST} (${TEST_VERSION})"
    echo "#"

    echo "% start_test" >> ${TESTSPEC}
    echo "\section{$CURRENT_TEST}" >> ${TESTSPEC}
}

# End test block. Each start_test should have a corresponding end_test call.
function end_test {
    echo "## ${TEST_COUNT} ${CURRENT_TEST} completed"
    echo "#####################################################################"
    CURRENT_TEST="none"
    end_table_if_step
    echo "% end_test" >> ${TESTSPEC}

    IN_STEP=0
}

function begin_table_if_not_step {
     if [ "${IN_STEP}" -eq "0" ]; then
        cat >> ${TESTSPEC} <<EOD
\begin{longtable}[H]{ABC}
\toprule
\textbf{Step} & \textbf{Description} & \textbf{Notes} \\\\ \hline
\endhead
EOD
    fi
}
function end_table_if_step {
     if [ "${IN_STEP}" -eq "1" ]; then
        cat >> ${TESTSPEC} <<EOD
\\\\
\end{longtable}
% end_table_if_step
EOD
     fi
}
# Start a new section block. This increments the section counter and resets the
# step counter.
function section {
    let SECTION_COUNT=SECTION_COUNT+1
    msg=$1;
    echo "====================================================="
    echo "= ${TEST_COUNT}.${SECTION_COUNT}: $msg"
    # This will show up in the request logs on the core service and makes it
    # easier to trace the tests.
    curl -s $API_ENDPOINT/test/section/${TEST_COUNT}.${SECTION_COUNT} > /dev/null
    STEP_COUNT=1
    end_table_if_step
    IN_STEP=0
    echo "% section" >> ${TESTSPEC}
    echo "\subsection{$msg}" >> ${TESTSPEC}
}

# Run a single step in a test. This is typically just a single operation.
function step {
    STEP_MSG=$1;
    if [ "${IN_STEP}" -eq "1" ]; then
        echo "\\\\" >> ${TESTSPEC}
    fi
    echo "-----------------------------------------------------"
    echo "- ${TEST_COUNT}.${SECTION_COUNT}.${STEP_COUNT}: ${STEP_MSG} "
    # This will show up in the request logs on the core service and makes it
    # easier to trace the tests.
    curl -s $API_ENDPOINT/test/section/${TEST_COUNT}.${SECTION_COUNT}.${STEP_COUNT} > /dev/null

    begin_table_if_not_step
    IN_STEP=1
    cat >> ${TESTSPEC} <<EOD
% step
${TEST_COUNT}.${SECTION_COUNT}.${STEP_COUNT} & ${STEP_MSG} &
EOD

    let STEP_COUNT=STEP_COUNT+1
}

# Add description block to the test, section or step. This can be used to give
# additional information on each step.
function description {
    DESC=$1

    echo "$DESC"

    echo "% description" >> ${TESTSPEC}

    echo ${DESC} >> ${TESTSPEC}
}

function test_version {
    TEST_VERSION=$1
    echo "Version: ${TEST_VERSION}"
}

failed_message() {
    cat << BOOM

                             ...----....
                         ..-:"''         ''"-..
                      .-'                      '-.
                    .'              .     .       '.
                  .'   .          .    .      .    .''.
                .'  .    .       .   .   .     .   . ..:.
              .' .   . .  .       .   .   ..  .   . ....::.
             ..   .   .      .  .    .     .  ..  . ....:IA.
            .:  .   .    .    .  .  .    .. .  .. .. ....:IA.
           .: .   .   ..   .    .     . . .. . ... ....:.:VHA.
           '..  .  .. .   .       .  . .. . .. . .....:.::IHHB.
          .:. .  . .  . .   .  .  . . . ...:.:... .......:HIHMM.
         .:.... .   . ."::"'.. .   .  . .:.:.:II;,. .. ..:IHIMMA
         ':.:..  ..::IHHHHHI::. . .  ...:.::::.,,,. . ....VIMMHM
        .:::I. .AHHHHHHHHHHAI::. .:...,:IIHHHHHHMMMHHL:. . VMMMM
       .:.:V.:IVHHHHHHHMHMHHH::..:" .:HIHHHHHHHHHHHHHMHHA. .VMMM.
       :..V.:IVHHHHHMMHHHHHHHB... . .:VPHHMHHHMMHHHHHHHHHAI.:VMMI
       ::V..:VIHHHHHHMMMHHHHHH. .   .I":IIMHHMMHHHHHHHHHHHAPI:WMM
       ::". .:.HHHHHHHHMMHHHHHI.  . .:..I:MHMMHHHHHHHHHMHV:':H:WM
       :: . :.::IIHHHHHHMMHHHHV  .ABA.:.:IMHMHMMMHMHHHHV:'. .IHWW
       '.  ..:..:.:IHHHHHMMHV" .AVMHMA.:.'VHMMMMHHHHHV:' .  :IHWV
        :.  .:...:".:.:TPP"   .AVMMHMMA.:. "VMMHHHP.:... .. :IVAI
       .:.   '... .:"'   .   ..HMMMHMMMA::. ."VHHI:::....  .:IHW'
       ...  .  . ..:IIPPIH: ..HMMMI.MMMV:I:.  .:ILLH:.. ...:I:IM
     : .   .'"' .:.V". .. .  :HMMM:IMMMI::I. ..:HHIIPPHI::'.P:HM.
     :.  .  .  .. ..:.. .    :AMMM IMMMM..:...:IV":T::I::.".:IHIMA
     'V:.. .. . .. .  .  .   'VMMV..VMMV :....:V:.:..:....::IHHHMH
       "IHH:.II:.. .:. .  . . . " :HB"" . . ..PI:.::.:::..:IHHMMV"
        :IP""HHII:.  .  .    . . .'V:. . . ..:IH:.:.::IHIHHMMMMM"
        :V:. VIMA:I..  .     .  . .. . .  .:.I:I:..:IHHHHMMHHMMM
        :"VI:.VWMA::. .:      .   .. .:. ..:.I::.:IVHHHMMMHMMMMI
        :."VIIHHMMA:.  .   .   .:  .:.. . .:.II:I:AMMMMMMHMMMMMI
        :..VIHIHMMMI...::.,:.,:!"I:!"I!"I!"V:AI:VAMMMMMMHMMMMMM'
        ':.:HIHIMHHA:"!!"I.:AXXXVVXXXXXXXA:."HPHIMMMMHHMHMMMMMV
          V:H:I:MA:W'I :AXXXIXII:IIIISSSSSSXXA.I.VMMMHMHMMMMMM
            'I::IVA ASSSSXSSSSBBSBMBSSSSSSBBMMMBS.VVMMHIMM'"'
             I:: VPAIMSSSSSSSSSBSSSMMBSSSBBMMMMXXI:MMHIMMI
            .I::. "H:XIIXBBMMMMMMMMMMMMMMMMMBXIXXMMPHIIMM'
            :::I.  ':XSSXXIIIIXSSBMBSSXXXIIIXXSMMAMI:.IMM
            :::I:.  .VSSSSSISISISSSBII:ISSSSBMMB:MI:..:MM
            ::.I:.  ':"SSSSSSSISISSXIIXSSSSBMMB:AHI:..MMM.
            ::.I:. . ..:"BBSSSSSSSSSSSSBBBMMMB:AHHI::.HMMI
            :..::.  . ..::":BBBBBSSBBBMMMB:MMMMHHII::IHHMI
            ':.I:... ....:IHHHHHMMMMMMMMMMMMMMMHHIIIIHMMV"
              "V:. ..:...:.IHHHMMMMMMMMMMMMMMMMHHHMHHMHP'
               ':. .:::.:.::III::IHHHHMMMMMHMHMMHHHHM"
                 "::....::.:::..:..::IIIIIHHHHMMMHHMV"
                   "::.::.. .. .  ...:::IIHHMMMMHMV"
                     "V::... . .I::IHHMMV"'
                       '"VHVHHHAHHHHMMV:"'
BOOM
    echo "**** Failed at test ${TEST_COUNT} ${CURRENT_TEST} -- ${SECTION_COUNT}.${STEP_COUNT}: ${STEP_MSG}"
    if [ -f ${TESTSPEC} ]; then
        rm ${TESTSPEC}
    fi
}

trap 'failed_message' ERR
