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
    stage('Image Build') {
      steps {
        sh './bin/build latest'
      }
    }
    stage('Test Sidecar Injector'){
      steps {
        sh 'summon -f ./tests/secrets.yml ./run-tests'
      }
    }

    stage('Publish Sidecar Injector Images') {
      when {
        branch 'master'
      }

      steps {
        sh './bin/publish latest'
      }
    }

  }

  post {
    always {
      cleanupAndNotify(currentBuild.currentResult)
    }
  }
}
