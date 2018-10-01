#!/usr/bin/env groovy

pipeline {
  agent { label 'executor-v2' }

  options {
    timestamps()
    buildDiscarder(logRotator(numToKeepStr: '30'))
  }

  stages {
    stage('Image Build') {
      steps {
        sh './bin/build latest'
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
