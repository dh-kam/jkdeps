package main

func failOnErrorExitCode(failOnError bool, failedFiles int) int {
	if failOnError && failedFiles > 0 {
		return 1
	}
	return 0
}
