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
                docker { image 'golang:1.15'
                         reuseNode true
                }
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

        stage('golangci-lint') {
            agent {
                docker { image 'golangci/golangci-lint:v1.40.0'
                         args '-u root --entrypoint=\'\''
                         reuseNode true
                }
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
