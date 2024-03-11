#!/usr/bin/env groovy

pipeline {
  agent { label 'conjur-enterprise-common-agent' }

  options {
    timestamps()
    buildDiscarder(logRotator(numToKeepStr: '30'))
  }

  triggers {
    cron(getDailyCronString())
  }

  stages {
    stage('Scan for internal URLs') {
      steps {
        script {
          detectInternalUrls()
        }
      }
    }
    
    stage('Get InfraPool ExecutorV2 Agent') {
      steps {
        script {
          // Request ExecutorV2 agents for 1 hour(s)
          INFRAPOOL_EXECUTORV2_AGENT_0 = getInfraPoolAgent.connected(type: "ExecutorV2", quantity: 1, duration: 1)[0]
        }
      }
    }

    stage('Validate') {
      parallel {
        stage('Changelog') {
          steps {
            script {
              INFRAPOOL_EXECUTORV2_AGENT_0.agentSh 'docker run --rm --volume "${PWD}/CHANGELOG.md":/CHANGELOG.md cyberark/parse-a-changelog'
            }
          }
        }
      }
    }

    stage('Image Build') {
      steps {
        script {
          INFRAPOOL_EXECUTORV2_AGENT_0.agentSh './bin/build'
        }
      }
    }
    stage('Test Sidecar Injector'){
      steps {
        script {
          INFRAPOOL_EXECUTORV2_AGENT_0.agentSh 'summon -f ./tests/secrets.yml ./run-tests'
        }
      }
    }

    stage('Scan Image') {
      parallel {
        stage("Scan image for fixable issues") {
          steps {
            scanAndReport(INFRAPOOL_EXECUTORV2_AGENT_0, "sidecar-injector:latest", "HIGH", false)
          }
        }
        stage("Scan image for all issues") {
          steps {
            scanAndReport(INFRAPOOL_EXECUTORV2_AGENT_0, "sidecar-injector:latest", "NONE", true)
          }
        }
      }
    }


    stage('Publish Edge Sidecar Injector Images') {
      when {
        branch 'main'
      }

      steps {
        script {
        INFRAPOOL_EXECUTORV2_AGENT_0.agentSh './bin/publish edge'
        }
      }
    }

    stage('Release') {
      // Only run this stage when triggered by a tag
      when { buildingTag() }

      parallel {
        stage('Publish Tagged Sidecar Injector Images') {
          steps {
            script {
            // The tag trigger sets TAG_NAME as an environment variable
            INFRAPOOL_EXECUTORV2_AGENT_0.agentSh './bin/publish'
            }
          }
        }
        stage('Create draft release') {
          steps {
            script {
            INFRAPOOL_EXECUTORV2_AGENT_0.agentSh "summon --yaml 'GITHUB_TOKEN: !var github/users/conjur-jenkins/api-token' ./bin/build_release"
            }
          }
        }
      }
    }

  }

  post {
    always {
      script {
        releaseInfraPoolAgent(".infrapool/release_agents")
      }
    }
  }
}
