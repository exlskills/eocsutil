var _ = require('lodash');
var JSDOM = require("jsdom").JSDOM;
var $ = require("jquery")(new JSDOM().window);

// Adapted from browser xblock: https://github.com/edx/edx-platform/blob/master/common/lib/xmodule/xmodule/js/src/problem/edit.js#L307
// At commit b370fe23148b3c6c38f9481e6658916f11fa0985
var markdownToXml = function(markdown) {
    var demandHintTags = [],
        finalDemandHints, finalXml, responseTypesMarkdown, responseTypesXML, toXml;

    escapeInnerMD = function(inner) {
        return inner.replace(/(?<=`.*)(\|)(?=.*`)/g, '&pipe;').replace(/(?<=`.*)(&)(?=.*`)/g, '&amp;').replace(/(?<=`.*)(<)(?=.*`)/g, '&lt;').replace(/(?<=`.*)(>)(?=.*`)/g, '&gt;').replace(/(?<=```.*)(\|)(?=.*```)/gms, '&pipe;').replace(/(?<=```.*)(&)(?=.*```)/gms, '&amp;').replace(/(?<=```.*)(<)(?=.*```)/gms, '&lt;').replace(/(?<=```.*)(>)(?=.*```)/gms, '&gt;');
    };

    toXml = function(partialMarkdown) {
        var xml = escapeInnerMD(partialMarkdown),
            i, splits, makeParagraph, serializer, responseType, $xml, responseTypesSelector,
            inputtype, beforeInputtype, extractHint, demandhints;
        var responseTypes = [
            'optionresponse', 'multiplechoiceresponse', 'stringresponse', 'numericalresponse', 'choiceresponse'
        ];

        // fix DOS \r\n line endings to look like \n
        xml = xml.replace(/\r\n/g, '\n');

        // replace headers
        xml = xml.replace(/(^.*?$)(?=\n\=\=+$)/gm, '<h3 class="hd hd-2 problem-header">$1</h3>');
        xml = xml.replace(/\n^\=\=+$/gm, '');

        // extract question and description(optional)
        // >>question||description<< converts to
        // <label>question</label> <description>description</description>
        xml = xml.replace(/>>([^]+?)<</gm, function(match, questionText) {
            var result = questionText.split('||'),
                label = '<label>' + result[0] + '</label>\n';

            // don't add empty <description> tag
            if (result.length === 1 || !result[1]) {
                return label;
            }
            return label + '<description>' + result[1] + '</description>\n';
        });

        // Pull out demand hints,  || a hint ||
        demandhints = '';
        xml = xml.replace(/(^\s*\|\|.*?\|\|\s*$\n?)+/gm, function(match) {  // $\n
            var inner,
                options = match.split('\n');
            for (i = 0; i < options.length; i += 1) {
                inner = /\s*\|\|(.*?)\|\|/.exec(options[i]);
                if (inner) {
                    // xss-lint: disable=javascript-concat-html
                    demandhints += '  <hint>' + inner[1].trim() + '</hint>\n';
                }
            }
            return '';
        });

        // replace \n+whitespace within extended hint {{ .. }}, by a space, so the whole
        // hint sits on one line.
        // This is the one instance of {{ ... }} matching that permits \n
        xml = xml.replace(/{{(.|\n)*?}}/gm, function(match) {
            return match.replace(/\r?\n( |\t)*/g, ' ');
        });

        // Function used in many places to extract {{ label:: a hint }}.
        // Returns a little hash with various parts of the hint:
        // hint: the hint or empty, nothint: the rest
        // labelassign: javascript assignment of label attribute, or empty
        extractHint = function(inputText, detectParens) {
            var text = inputText,
                curly = /\s*{{(.*?)}}/.exec(text),
                hint = '',
                label = '',
                parens = false,
                labelassign = '',
                labelmatch;
            if (curly) {
                text = text.replace(curly[0], '');
                hint = curly[1].trim();
                labelmatch = /^(.*?)::/.exec(hint);
                if (labelmatch) {
                    hint = hint.replace(labelmatch[0], '').trim();
                    label = labelmatch[1].trim();
                    labelassign = ' label="' + label + '"';
                }
            }
            if (detectParens) {
                if (text.length >= 2 && text[0] === '(' && text[text.length - 1] === ')') {
                    text = text.substring(1, text.length - 1);
                    parens = true;
                }
            }
            return {
                nothint: text,
                hint: hint,
                label: label,
                parens: parens,
                labelassign: labelassign
            };
        };


        // replace selects
        // [[ a, b, (c) ]]
        // [[
        //     a
        //     b
        //     (c)
        //  ]]
        // <optionresponse>
        //  <optioninput>
        //     <option  correct="True">AAA<optionhint  label="Good Job">
        //          Yes, multiple choice is the right answer.
        //  </optionhint>
        // Note: part of the option-response syntax looks like multiple-choice, so it must be processed first.
        xml = xml.replace(/\[\[((.|\n)+?)\]\]/g, function(match, group1) {
            var textHint, options, optiontag, correct, lines, optionlines, line, correctstr, hintstr, label;
            // decide if this is old style or new style
            if (match.indexOf('\n') === -1) {  // OLD style, [[ .... ]]  on one line
                options = group1.split(/\,\s*/g);
                optiontag = '  <optioninput options="(';
                for (i = 0; i < options.length; i += 1) {
                    optiontag += "'" + options[i].replace(/(?:^|,)\s*\((.*?)\)\s*(?:$|,)/g, '$1') + "'" +
                        (i < options.length - 1 ? ',' : '');
                }
                optiontag += ')" correct="';
                correct = /(?:^|,)\s*\((.*?)\)\s*(?:$|,)/g.exec(group1);
                if (correct) {
                    optiontag += correct[1];
                }
                optiontag += '">';
                return '\n<optionresponse>\n' + optiontag + '</optioninput>\n</optionresponse>\n\n';
            }

            // new style  [[ many-lines ]]
            lines = group1.split('\n');
            optionlines = '';
            for (i = 0; i < lines.length; i++) {
                line = lines[i].trim();
                if (line.length > 0) {
                    textHint = extractHint(line, true);
                    correctstr = ' correct="' + (textHint.parens ? 'True' : 'False') + '"';
                    hintstr = '';
                    if (textHint.hint) {
                        label = textHint.label;
                        if (label) {
                            label = ' label="' + label + '"';
                        }
                        hintstr = ' <optionhint' + label + '>' + textHint.hint + '</optionhint>';
                    }
                    optionlines += '    <option' + correctstr + '>' + textHint.nothint + hintstr +
                        '</option>\n';
                }
            }
            return '\n<optionresponse>\n  <optioninput>\n' + optionlines +
                '  </optioninput>\n</optionresponse>\n\n';
        });

        // multiple choice questions
        //
        xml = xml.replace(/(^\s*\(.{0,3}\).*?$\n*)+/gm, function(match) {
            var choices = '',
                shuffle = false,
                options = match.split('\n'),
                value, inparens, correct,
                fixed, hint, result;
            for (i = 0; i < options.length; i++) {
                options[i] = options[i].trim();                   // trim off leading/trailing whitespace
                if (options[i].length > 0) {
                    value = options[i].split(/^\s*\(.{0,3}\)\s*/)[1];
                    inparens = /^\s*\((.{0,3})\)\s*/.exec(options[i])[1];
                    correct = /x/i.test(inparens);
                    fixed = '';
                    if (/@/.test(inparens)) {
                        fixed = ' fixed="true"';
                    }
                    if (/!/.test(inparens)) {
                        shuffle = true;
                    }

                    hint = extractHint(value);
                    if (hint.hint) {
                        value = hint.nothint;
                        value = value + ' <choicehint' + hint.labelassign + '>' + hint.hint + '</choicehint>';
                    }
                    choices += '    <choice correct="' + correct + '"' + fixed + '>' + value + '</choice>\n';
                }
            }
            result = '<multiplechoiceresponse>\n';
            if (shuffle) {
                result += '  <choicegroup type="MultipleChoice" shuffle="true">\n';
            } else {
                result += '  <choicegroup type="MultipleChoice">\n';
            }
            result += choices;
            result += '  </choicegroup>\n';
            result += '</multiplechoiceresponse>\n\n';
            return result;
        });

        // TODO propogate this multi-line awesomeness to other question types
        xml = xml.replace(/(^\s*\+\(.{0,3}\)(\n|.(?<!\-\(.{0,3}\)\-))*$\n*)+/gm, function(match) {
            var choices = '',
                shuffle = false,
                // @svarlamov, originally this was splitting by \n
                options = match.split(/(^\s*\+\(.{0,3}\)(\n|.(?<!\-\(.{0,3}\)))*$\n*)+/gm),
                value, inparens, correct,
                fixed, hint, result;
            for (i = 0; i < options.length; i++) {
                options[i] = options[i].trim();                   // trim off leading/trailing whitespace
                options[i] = options[i].replace(/^\s*-\(.{0,3}\)\s*/, '');
                if (options[i].length > 0) {
                    value = options[i].split(/^\s*\+\(.{0,3}\)\s*/)[1];
                    inparens = /^\s*\+\((.{0,3})\)\s*/.exec(options[i])[1];
                    correct = /x/i.test(inparens);
                    fixed = '';
                    if (/@/.test(inparens)) {
                        fixed = ' fixed="true"';
                    }
                    if (/!/.test(inparens)) {
                        shuffle = true;
                    }

                    hint = extractHint(value);
                    if (hint.hint) {
                        value = hint.nothint;
                        value = value + ' <choicehint' + hint.labelassign + '>' + hint.hint + '</choicehint>';
                    }
                    choices += '    <choice correct="' + correct + '"' + fixed + '>' + value + '</choice>\n';
                }
            }
            result = '<multiplechoiceresponse>\n';
            if (shuffle) {
                result += '  <choicegroup type="MultipleChoice" shuffle="true">\n';
            } else {
                result += '  <choicegroup type="MultipleChoice">\n';
            }
            result += choices;
            result += '  </choicegroup>\n';
            result += '</multiplechoiceresponse>\n\n';
            return result;
        });

        // group check answers
        // [.] with {{...}} lines mixed in
        xml = xml.replace(/(^\s*((\[.?\])|({{.*?}})).*?$\n*)+/gm, function(match) {
            var groupString = '<choiceresponse>\n',
                options = match.split('\n'),
                value, correct, abhint, endHints, hintbody,
                hint, inner, select, hints;

            groupString += '  <checkboxgroup>\n';
            endHints = '';  // save these up to emit at the end

            for (i = 0; i < options.length; i += 1) {
                if (options[i].trim().length > 0) {
                    // detect the {{ ((A*B)) ...}} case first
                    // emits: <compoundhint value="A*B">AB hint</compoundhint>

                    abhint = /^\s*{{\s*\(\((.*?)\)\)(.*?)}}/.exec(options[i]);
                    if (abhint) {
                        // lone case of hint text processing outside of extractHint, since syntax here is unique
                        hintbody = abhint[2];
                        hintbody = hintbody.replace('&lf;', '\n').trim();
                        endHints += '    <compoundhint value="' + abhint[1].trim() + '">' + hintbody +
                            '</compoundhint>\n';
                        continue;  // bail
                    }

                    value = options[i].split(/^\s*\[.?\]\s*/)[1];
                    correct = /^\s*\[x\]/i.test(options[i]);
                    hints = '';
                    //  {{ selected: You’re right that apple is a fruit. },
                    //   {unselected: Remember that apple is also a fruit.}}
                    hint = extractHint(value);
                    if (hint.hint) {
                        inner = '{' + hint.hint + '}';  // parsing is easier if we put outer { } back

                        // include \n since we are downstream of extractHint()
                        select = /{\s*(s|selected):((.|\n)*?)}/i.exec(inner);
                        // checkbox choicehints get their own line, since there can be two of them
                        // <choicehint selected="true">You’re right that apple is a fruit.</choicehint>
                        if (select) {
                            hints += '\n      <choicehint selected="true">' + select[2].trim() +
                                '</choicehint>';
                        }
                        select = /{\s*(u|unselected):((.|\n)*?)}/i.exec(inner);
                        if (select) {
                            hints += '\n      <choicehint selected="false">' + select[2].trim() +
                                '</choicehint>';
                        }

                        // Blank out the original text only if the specific "selected" syntax is found
                        // That way, if the user types it wrong, at least they can see it's not processed.
                        if (hints) {
                            value = hint.nothint;
                        }
                    }
                    groupString += '    <choice correct="' + correct + '">' + value + hints + '</choice>\n';
                }
            }

            groupString += endHints;
            groupString += '  </checkboxgroup>\n';
            groupString += '</choiceresponse>\n\n';

            return groupString;
        });


        // replace string and numerical, numericalresponse, stringresponse
        // A fine example of the function-composition programming style.
        xml = xml.replace(/(^s?\=\s*(.*?$)(\n*(or|not)\=\s*(.*?$))*)+/gm, function(match, p) {
            // Line split here, trim off leading xxx= in each function
            var answersList = p.split('\n'),

                isRangeToleranceCase = function(answer) {
                    return _.includes(
                        ['[', '('], answer[0]) && _.includes([']', ')'], answer[answer.length - 1]
                    );
                },

                getAnswerData = function(answerValue) {
                    var answerData = {},
                        answerParams = /(.*?)\+\-\s*(.*?$)/.exec(answerValue);
                    if (answerParams) {
                        answerData.answer = answerParams[1].replace(/\s+/g, ''); // inputs like 5*2 +- 10
                        answerData.default = answerParams[2];
                    } else {
                        answerData.answer = answerValue.replace(/\s+/g, ''); // inputs like 5*2
                    }
                    return answerData;
                },

                processNumericalResponse = function(answerValues) {
                    var firstAnswer, answerData, numericalResponseString, additionalAnswerString,
                        textHint, hintLine, additionalTextHint, additionalHintLine, orMatch, hasTolerance;

                    // First string case is s?= [e.g. = 100]
                    firstAnswer = answerValues[0].replace(/^\=\s*/, '');

                    // If answer is not numerical
                    if (isNaN(parseFloat(firstAnswer)) && !isRangeToleranceCase(firstAnswer)) {
                        return false;
                    }

                    textHint = extractHint(firstAnswer);
                    hintLine = '';
                    if (textHint.hint) {
                        firstAnswer = textHint.nothint;
                        // xss-lint: disable=javascript-concat-html
                        hintLine = '  <correcthint' + textHint.labelassign + '>' +
                            // xss-lint: disable=javascript-concat-html
                            textHint.hint + '</correcthint>\n';
                    }

                    // Range case
                    if (isRangeToleranceCase(firstAnswer)) {
                        // [5, 7) or (5, 7), or (1.2345 * (2+3), 7*4 ]  - range tolerance case
                        // = (5*2)*3 should not be used as range tolerance
                        // xss-lint: disable=javascript-concat-html
                        numericalResponseString = '<numericalresponse answer="' + firstAnswer + '">\n';
                    } else {
                        answerData = getAnswerData(firstAnswer);
                        // xss-lint: disable=javascript-concat-html
                        numericalResponseString = '<numericalresponse answer="' + answerData.answer + '">\n';
                        if (answerData.default) {
                            // xss-lint: disable=javascript-concat-html
                            numericalResponseString += '  <responseparam type="tolerance" default="' +
                                // xss-lint: disable=javascript-concat-html
                                answerData.default + '" />\n';
                        }
                    }

                    // Additional answer case or= [e.g. or= 10]
                    // Since answerValues[0] is firstAnswer, so we will not include this in additional answers.
                    additionalAnswerString = '';
                    for (i = 1; i < answerValues.length; i++) {
                        additionalHintLine = '';
                        additionalTextHint = extractHint(answerValues[i]);
                        orMatch = /^or\=\s*(.*)/.exec(additionalTextHint.nothint);
                        if (orMatch) {
                            hasTolerance = /(.*?)\+\-\s*(.*?$)/.exec(orMatch[1]);
                            // Do not add additional_answer if additional answer is not numerical (eg. or= ABC)
                            // or contains range tolerance case (eg. or= (5,7)
                            // or has tolerance (eg. or= 10 +- 0.02)
                            if (isNaN(parseFloat(orMatch[1])) ||
                                isRangeToleranceCase(orMatch[1]) ||
                                hasTolerance) {
                                continue;
                            }

                            if (additionalTextHint.hint) {
                                // xss-lint: disable=javascript-concat-html
                                additionalHintLine = '<correcthint' +
                                    // xss-lint: disable=javascript-concat-html
                                    additionalTextHint.labelassign + '>' +
                                    // xss-lint: disable=javascript-concat-html
                                    additionalTextHint.hint + '</correcthint>';
                            }

                            // xss-lint: disable=javascript-concat-html
                            additionalAnswerString += '  <additional_answer answer="' + orMatch[1] + '">';
                            additionalAnswerString += additionalHintLine;
                            additionalAnswerString += '</additional_answer>\n';
                        }
                    }

                    // Add additional answers string to numerical problem string.
                    if (additionalAnswerString) {
                        numericalResponseString += additionalAnswerString;
                    }

                    numericalResponseString += '  <formulaequationinput />\n';
                    numericalResponseString += hintLine;
                    numericalResponseString += '</numericalresponse>\n\n';

                    return numericalResponseString;
                },

                processStringResponse = function(values) {
                    var firstAnswer, textHint, typ, string, orMatch, notMatch;
                    // First string case is s?=
                    firstAnswer = values.shift();
                    firstAnswer = firstAnswer.replace(/^s?\=\s*/, '');
                    textHint = extractHint(firstAnswer);
                    firstAnswer = textHint.nothint;
                    typ = ' type="ci"';
                    if (firstAnswer[0] === '|') { // this is regexp case
                        typ = ' type="ci regexp"';
                        firstAnswer = firstAnswer.slice(1).trim();
                    }
                    string = '<stringresponse answer="' + firstAnswer + '"' + typ + ' >\n';
                    if (textHint.hint) {
                        string += '  <correcthint' + textHint.labelassign + '>' +
                            textHint.hint + '</correcthint>\n';
                    }

                    // Subsequent cases are not= or or=
                    for (i = 0; i < values.length; i += 1) {
                        textHint = extractHint(values[i]);
                        notMatch = /^not\=\s*(.*)/.exec(textHint.nothint);
                        if (notMatch) {
                            string += '  <stringequalhint answer="' + notMatch[1] + '"' +
                                textHint.labelassign + '>' + textHint.hint + '</stringequalhint>\n';
                            continue;
                        }
                        orMatch = /^or\=\s*(.*)/.exec(textHint.nothint);
                        if (orMatch) {
                            // additional_answer with answer= attribute
                            string += '  <additional_answer answer="' + orMatch[1] + '">';
                            if (textHint.hint) {
                                string += '<correcthint' + textHint.labelassign + '>' +
                                    textHint.hint + '</correcthint>';
                            }
                            string += '</additional_answer>\n';
                        }
                    }

                    string += '  <textline size="20"/>\n</stringresponse>\n\n';

                    return string;
                };

            return processNumericalResponse(answersList) || processStringResponse(answersList);
        });

        // replace explanations
        xml = xml.replace(/\[explanation\]\n?([^\]]*)\[\/?explanation\]/gmi, function(match, p1) {
            return '<solution>\n<div class="detailed-solution">\n' +
                gettext('Explanation') + '\n\n' + p1 + '\n</div>\n</solution>';
        });

        // replace code blocks
        xml = xml.replace(/\[code\]\n?([^\]]*)\[\/?code\]/gmi, function(match, p1) {
            return '<pre><code>' + p1 + '</code></pre>';
        });

        // split scripts and preformatted sections, and wrap paragraphs
        splits = xml.split(/(\<\/?(?:script|pre|label|description).*?\>)/g);

        // Wrap a string by <p> tag when line is not already wrapped by another tag
        // true when line is not already wrapped by another tag false otherwise
        makeParagraph = true;

        for (i = 0; i < splits.length; i += 1) {
            if (/\<(script|pre|label|description)/.test(splits[i])) {
                makeParagraph = false;
            }

            if (makeParagraph) {
                splits[i] = splits[i].replace(/(^(?!\s*\<|$).*$)/gm, '<p>$1</p>');
            }

            if (/\<\/(script|pre|label|description)/.test(splits[i])) {
                makeParagraph = true;
            }
        }

        xml = splits.join('');

        // rid white space
        xml = xml.replace(/\n\n\n/g, '\n');

        // if we've come across demand hints, wrap in <demandhint> at the end
        if (demandhints) {
            demandHintTags.push(demandhints);
        }

        // make selector to search responsetypes in xml
        responseTypesSelector = responseTypes.join(', ');

        // make temporary xml
        // xss-lint: disable=javascript-concat-html
        $xml = $($.parseXML('<prob>' + xml + '</prob>'));
        responseType = $xml.find(responseTypesSelector);

        // convert if there is only one responsetype
        if (responseType.length === 1) {
            inputtype = responseType[0].firstElementChild;
            // used to decide whether an element should be placed before or after an inputtype
            beforeInputtype = true;

            _.each($xml.find('prob').children(), function(child) {
                // we don't want to add the responsetype again into new xml
                if (responseType[0].nodeName === child.nodeName) {
                    beforeInputtype = false;
                    return;
                }

                if (beforeInputtype) {
                    // xss-lint: disable=javascript-jquery-insert-into-target
                    responseType[0].insertBefore(child, inputtype);
                } else {
                    responseType[0].appendChild(child);
                }
            });

            // Note: replaced with 'xmlserializer' dep, as there is no native XMLSerializer in node
            // serializer = new XMLSerializer();
            var serializer = require('xmlserializer');
            xml = serializer.serializeToString(responseType[0]);

            // remove xmlns attribute added by the serializer
            xml = xml.replace(/\sxmlns=['"].*?['"]/gi, '');

            // XMLSerializer messes the indentation of XML so add newline
            // at the end of each ending tag to make the xml looks better
            xml = xml.replace(/(\<\/.*?\>)(\<.*?\>)/gi, '$1\n$2');
        }

        // remove class attribute added on <p> tag for question title
        xml = xml.replace(/\sclass=\'qtitle\'/gi, '');
        return xml;
    };
    responseTypesXML = [];
    responseTypesMarkdown = markdown.split(/\n\s*---\s*\n/g);
    _.each(responseTypesMarkdown, function(responseTypeMarkdown) {
        if (responseTypeMarkdown.trim().length > 0) {
            responseTypesXML.push(toXml(responseTypeMarkdown));
        }
    });
    finalDemandHints = '';
    if (demandHintTags.length) {
        // xss-lint: disable=javascript-concat-html
        finalDemandHints = '\n<demandhint>\n' + demandHintTags.join('') + '</demandhint>';
    }
    // make all responsetypes descendants of a single problem element
    // xss-lint: disable=javascript-concat-html
    finalXml = '<problem>\n' + responseTypesXML.join('\n\n') + finalDemandHints + '\n</problem>';
    return finalXml;
};

module.exports = markdownToXml;
