#!groovy

pipeline {
    agent any

    options {
        ansiColor('xterm')
        timestamps()
    }

    stages {
        stage('Go Code Formatting') {
            agent {
                docker { image 'golang:1.10' }
            }
            steps {
		sh script: "./gofmt.sh"
            }
        }

        stage('DRAM') {
            steps {
                sh "dram --username jenkins -d yang"
            }
        }
        stage('golint') {
            agent {
                docker { image 'golang:1.10' }
            }
            steps {
                sh script: "go get -u golang.org/x/lint/golint"
                sh script: "golint -set_exit_status ./... > golint.txt", returnStatus: true
            }

            post {
                always {
                    recordIssues tool: goLint(pattern: 'golint.txt'),
                        qualityGates: [[type: 'TOTAL', threshold: 1, unstable: true]]

                    deleteDir()
                }
            }
        }

        stage('golangci-lint') {
            agent {
                docker { image 'golangci/golangci-lint:v1.10.2'
                         args '--entrypoint=\'\'' }
            }
            steps {
                // .golangci.yml can contain config on which checks to perform
                sh script: "golangci-lint run --out-format checkstyle ./... > checkstyle-result.xml",
                   returnStatus: true
            }

            post {
                always {
                    recordIssues tool: checkStyle(),
                        qualityGates: [[type: 'TOTAL', threshold: 1, unstable: true]]

                    deleteDir()
                }
            }
        }
    }

}
