package mailer

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func errorEquals(err error, expected error) bool {
	if err == nil || expected == nil {
		return err == expected
	}
	return fmt.Sprintf("%v", err) == fmt.Sprintf("%v", expected)
}

func TestErrorEquals(t *testing.T) {
	testCases := []struct {
		label string

		err      error
		expected error

		expectedResult bool
	}{
		{
			label:          "err is nil and expected is nil",
			err:            nil,
			expected:       nil,
			expectedResult: true,
		},
		{
			label:          "err message != expected message",
			err:            errors.New("x"),
			expected:       errors.New("y"),
			expectedResult: false,
		},
		{
			label:          "err message == expected message",
			err:            errors.New("x"),
			expected:       errors.New("x"),
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		res := errorEquals(tc.err, tc.expected)

		if res != tc.expectedResult {
			t.Errorf("%v: result got %v, want %v", tc.label, res, tc.expectedResult)
		}
	}
}

func errorMessageEquals(err error, expected string) bool {
	if err == nil {
		return expected == ""
	}
	errStr := fmt.Sprintf("%v", err)
	return errStr == expected
}

func TestErrorMessageEquals(t *testing.T) {
	testCases := []struct {
		label string

		err      error
		expected string

		expectedResult bool
	}{
		{
			label:          "err is nil and expected is empty",
			err:            nil,
			expected:       "",
			expectedResult: true,
		},
		{
			label:          "err is nil and expected is not empty",
			err:            nil,
			expected:       "x",
			expectedResult: false,
		},
		{
			label:          "err is not nil and expected is empty",
			err:            errors.New("x"),
			expected:       "",
			expectedResult: false,
		},
		{
			label:          "err is not nil and err message is not expected",
			err:            errors.New("x"),
			expected:       "y",
			expectedResult: false,
		},
		{
			label:          "err is not nil and err message is expected",
			err:            errors.New("x"),
			expected:       "x",
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		res := errorMessageEquals(tc.err, tc.expected)

		if res != tc.expectedResult {
			t.Errorf("%v: result got %v, want %v", tc.label, res, tc.expectedResult)
		}
	}
}

func errorMessageStartsWith(err error, prefix string) bool {
	if err == nil {
		return prefix == ""
	}
	if prefix == "" {
		return false
	}
	errStr := fmt.Sprintf("%v", err)
	return strings.Index(errStr, prefix) == 0
}

func TestErrorMessageStartsWith(t *testing.T) {
	testCases := []struct {
		label string

		err    error
		prefix string

		expected bool
	}{
		{
			label:    "err is nil and prefix is empty",
			err:      nil,
			prefix:   "",
			expected: true,
		},
		{
			label:    "err is nil and prefix is not empty",
			err:      nil,
			prefix:   "x",
			expected: false,
		},
		{
			label:    "err is not nil and prefix is empty",
			err:      errors.New("x"),
			prefix:   "",
			expected: false,
		},
		{
			label:    "err is not nil and err message is not expected",
			err:      errors.New("x: 1"),
			prefix:   "y",
			expected: false,
		},
		{
			label:    "err is not nil and err message is expected",
			err:      errors.New("x: 1"),
			prefix:   "x",
			expected: true,
		},
	}

	for _, tc := range testCases {
		res := errorMessageStartsWith(tc.err, tc.prefix)

		if res != tc.expected {
			t.Errorf("%v: result got %v, want %v", tc.label, res, tc.expected)
		}
	}
}
