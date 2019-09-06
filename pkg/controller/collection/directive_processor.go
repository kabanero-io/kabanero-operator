package collection

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// The DirectiveProcessor processes text processing directives found in the yaml source
type DirectiveProcessor struct {
}

func (g DirectiveProcessor) Render(b []byte, context map[string]interface{}) ([]byte, error) {
	//Search for directives
	directiveExpr := regexp.MustCompile(`\s?(#Kabanero!.*)$`)
	directives := make([]string, 0)
	reader := bufio.NewReader(bytes.NewReader(b))
	for {
		line, _, err := reader.ReadLine()

		if err == io.EOF {
			break
		}

		if directiveExpr.Match(line) {
			directive := directiveExpr.FindStringSubmatch(string(line))[1]
			directives = append(directives, directive)
		}
	}

	text := string(b)
	for _, directive := range directives {
		var err error
		text, err = g.process_directive(directive, text, context)
		if err != nil {
			return nil, err
		}
	}

	return []byte(text), nil
}

//process_directive processes an individual directive like: #Kabanero! on activate substitute CollectionName for text '${collection-name}'
func (g DirectiveProcessor) process_directive(directive string, text string, context map[string]interface{}) (string, error) {
	textSubstitutionExpr := regexp.MustCompile(`#Kabanero!\son\sactivate\s(substitute\s(.+?)\s(for text)\s'(.+?)')`)
	if textSubstitutionExpr.MatchString(directive) {
		groups := textSubstitutionExpr.FindStringSubmatch(directive)

		//group[0]: e.g. #Kabanero! on activate substitute CollectionName for text '${collection-name}'
		//group[1]: e.g. substitute CollectionName for text '${collection-name}'
		key := groups[2]               // The variable found in the context (CollectionName)
		substitution_type := groups[3] //e.g. for text

		if substitution_type == "for text" {
			text_to_replace := groups[4] //e.g. '${collection-name}'

			//Prune the directive from the text first
			text = strings.Replace(text, directive, "", 1)
			text = strings.TrimSpace(text)

			text = strings.ReplaceAll(text, text_to_replace, context[key].(string))

			return text, nil
		} else {
			return "", fmt.Errorf("Unknown substitution: %v", substitution_type)
		}
	} else {
		return "", fmt.Errorf("Unknown directive: %v", directive)
	}
}
