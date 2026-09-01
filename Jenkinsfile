library(
        identifier: 'jenkins-lib-common@v4.9.2',
        retriever: modernSCM([
                $class: 'GitSCMSource',
                credentialsId: 'jenkins-integration-with-github-account',
                remote: 'git@github.com:zextras/jenkins-lib-common.git',
        ])
)

properties(defaultPipelineProperties())

pipeline {
    agent {
        node {
            label 'zextras-v1'
        }
    }

    environment {
        GITHUB_BOT_PR_CREDS = credentials('jenkins-integration-with-github-account')
        JAVA_OPTS = '-Dfile.encoding=UTF8'
        LC_ALL = 'C.UTF-8'
    }

    options {
        buildDiscarder(logRotator(numToKeepStr: '25'))
        disableConcurrentBuilds()
        skipDefaultCheckout()
        timeout(time: 2, unit: 'HOURS')
    }

    triggers {
        cron(env.BRANCH_IS_PRIMARY == 'true' ? 'H 5 * * *' : '')
    }

    stages {
        stage('Setup') {
            steps {
                checkout scm
                gitMetadata()
            }
        }

        stage('Security Scan') {
            steps {
                gitleaksStage()
            }
        }

        stage('Build and Publish Docker images') {
            steps {
                dockerStage(images: [
                          [
                            dockerfile: 'images/Dockerfile-envoy',
                            imageName : 'carbonio-sidecar',
                            platforms : ['linux/amd64', 'linux/arm64'] as Set,
                            ocLabels  : [
                                    title : 'Carbonio Sidecar Container Base image',
                            ]
                          ],
                          [
                            dockerfile  : 'composed-ui/Dockerfile',
                            buildContext: 'composed-ui',
                            imageName   : 'carbonio-composed-ui',
                            platforms   : ['linux/amd64', 'linux/arm64'] as Set,
                            ocLabels    : [
                                    title: 'Carbonio Composed UI',
                            ]
                          ]
                        ])
            }
        }
    }
}
