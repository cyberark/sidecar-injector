#!/usr/bin/env groovy

pipeline {
  agent { label 'executor-v2' }

  options {
    timestamps()
    buildDiscarder(logRotator(numToKeepStr: '30'))
  }

  triggers {
    cron(getDailyCronString())
  }

  stages {
    stage('Validate') {
      parallel {
        stage('Changelog') {
          steps { sh 'docker run --rm --volume "${PWD}/CHANGELOG.md":/CHANGELOG.md cyberark/parse-a-changelog' }
        }
      }
    }

    stage('Image Build') {
      steps {
        sh './bin/build'
      }
    }
    stage('Test Sidecar Injector'){
      steps {
        sh 'summon -f ./tests/secrets.yml ./run-tests'
      }
    }

    stage('Scan Image') {
      parallel {
        stage("Scan image for fixable issues") {
          steps {
            scanAndReport("sidecar-injector:latest", "HIGH", false)
          }
        }
        stage("Scan image for all issues") {
          steps {
            scanAndReport("sidecar-injector:latest", "NONE", true)
          }
        }
      }
    }


    stage('Publish Edge Sidecar Injector Images') {
      when {
        branch 'master'
      }

      steps {
        sh './bin/publish edge'
      }
    }

    stage('Release') {
      // Only run this stage when triggered by a tag
      when { tag "v*" }

      parallel {
        stage('Publish Tagged Sidecar Injector Images') {
          steps {
            // The tag trigger sets TAG_NAME as an environment variable
            sh './bin/publish'
          }
        }
        stage('Create draft release') {
          steps {
            sh "summon --yaml 'GITHUB_TOKEN: !var github/users/conjur-jenkins/api-token' ./bin/build_release"
            archiveArtifacts 'dist/goreleaser/'
          }
        }
      }
    }

  }

  post {
    always {
      cleanupAndNotify(currentBuild.currentResult)
    }
  }
}
