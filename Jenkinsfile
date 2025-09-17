#!/usr/bin/env groovy
@Library(["product-pipelines-shared-library", "conjur-enterprise-sharedlib"]) _

// Automated release, promotion and dependencies
properties([
  // Include the automated release parameters for the build
  release.addParams(),
  // Dependencies of the project that should trigger builds
  dependencies([])
])

// Performs release promotion.  No other stages will be run
if (params.MODE == "PROMOTE") {
  release.promote(params.VERSION_TO_PROMOTE) { infrapool, sourceVersion, targetVersion, assetDirectory ->
    // Any assets from sourceVersion Github release are available in assetDirectory
    // Any version number updates from sourceVersion to targetVersion occur here
    // Any publishing of targetVersion artifacts occur here
    // Anything added to assetDirectory will be attached to the Github Release

    runSecurityScans(infrapool,
      image: "registry.tld/sidecar-injector:${sourceVersion}-${gitCommit(INFRAPOOL_EXECUTORV2_AGENT_0)}",
      buildMode: params.MODE,
      branch: env.BRANCH_NAME)

    INFRAPOOL_EXECUTORV2_AGENT_0.agentGet from: "${assetDirectory}/", to: "./"
    signArtifacts patterns: ["*.tar.gz"]
    INFRAPOOL_EXECUTORV2_AGENT_0.agentPut from: "*.sig", to: "${assetDirectory}"

    // Pull existing images from internal registry in order to promote
    infrapool.agentSh """
      export PATH="release-tools/bin:${PATH}"
      docker pull registry.tld/sidecar-injector:${sourceVersion}-${gitCommit(INFRAPOOL_EXECUTORV2_AGENT_0)}
      # Promote source version to target version.
      ./bin/publish --promote --source ${sourceVersion}-${gitCommit(INFRAPOOL_EXECUTORV2_AGENT_0)} --target ${targetVersion}
    """
    
    // Resolve ownership issue before promotion
    sh 'git config --global --add safe.directory ${PWD}'
  }

  // Copy Github Enterprise release to Github
  release.copyEnterpriseRelease(params.VERSION_TO_PROMOTE)
  return
}

pipeline {
  agent { label 'conjur-enterprise-common-agent' }

  options {
    timestamps()
    // We want to avoid running in parallel.
    // When we have 2 build running on the same environment (gke env only) in parallel,
    // we get the error "gcloud crashed : database is locked"
    disableConcurrentBuilds()
    buildDiscarder(logRotator(numToKeepStr: '30'))
    timeout(time: 3, unit: 'HOURS')
  }

  environment {
    // Sets the MODE to the specified or autocalculated value as appropriate
    MODE = release.canonicalizeMode()
  }

  triggers {
    cron(getDailyCronString())
    parameterizedCron(getWeeklyCronString("H(1-5)","%MODE=RELEASE"))
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

    // Aborts any builds triggered by another project that wouldn't include any changes
    stage ("Skip build if triggering job didn't create a release") {
      when {
        expression {
          MODE == "SKIP"
        }
      }
      steps {
        script {
          currentBuild.result = 'ABORTED'
          error("Aborting build because this build was triggered from upstream, but no release was built")
        }
      }
    }

    // Generates a VERSION file based on the current build number and latest version in CHANGELOG.md
    stage('Validate Changelog and set version') {
      steps {
        updateVersion(INFRAPOOL_EXECUTORV2_AGENT_0, "CHANGELOG.md", "${BUILD_NUMBER}")
      }
    }

    stage('Get latest upstream dependencies') {
      steps {
        script {
          updatePrivateGoDependencies("${WORKSPACE}/go.mod")
          // Copy the vendor directory onto infrapool
          INFRAPOOL_EXECUTORV2_AGENT_0.agentPut from: "vendor", to: "${WORKSPACE}"
          INFRAPOOL_EXECUTORV2_AGENT_0.agentPut from: "go.*", to: "${WORKSPACE}"
          // Add GOMODCACHE directory to infrapool allowing automated release to generate SBOMs
          INFRAPOOL_EXECUTORV2_AGENT_0.agentPut from: "/root/go", to: "/var/lib/jenkins/"
        }
      }
    }

    stage('Build while unit testing') {
      parallel {

        stage('Image Build') {
          steps {
            script {
              INFRAPOOL_EXECUTORV2_AGENT_0.agentSh './bin/build'
            }
          }
        }

        stage('Run unit tests') {
          steps {
            script {
              INFRAPOOL_EXECUTORV2_AGENT_0.agentSh './bin/test_unit'
            }
          }
          post {
            always {
              script {
                INFRAPOOL_EXECUTORV2_AGENT_0.agentSh './bin/coverage'
                INFRAPOOL_EXECUTORV2_AGENT_0.agentStash name: 'xml-out', includes: '*.xml'
                unstash 'xml-out'
                junit 'junit.xml'

                cobertura autoUpdateHealth: false,
                  autoUpdateStability: false,
                  coberturaReportFile: 'coverage.xml',
                  conditionalCoverageTargets: '70, 0, 0',
                  failUnhealthy: false,
                  failUnstable: false,
                  maxNumberOfBuilds: 0,
                  lineCoverageTargets: '70, 0, 0',
                  methodCoverageTargets: '70, 0, 0',
                  onlyStable: false,
                  sourceEncoding: 'ASCII',
                  zoomCoverageChart: false
                codacy action: 'reportCoverage', filePath: "coverage.xml"
              }
            }
          }
        }
      }
    }

    stage('Run Integration Tests'){
      steps {
        script {
          INFRAPOOL_EXECUTORV2_AGENT_0.agentSh 'summon -f ./tests/secrets.yml ./run-tests'
        }
      }
    }

    stage('Build release artifacts') {
      steps {
        script {
          INFRAPOOL_EXECUTORV2_AGENT_0.agentDir('./pristine-checkout') {
            // Go releaser requires a pristine checkout
            checkout scm

            // Copy the checkout content onto infrapool
            INFRAPOOL_EXECUTORV2_AGENT_0.agentPut from: "./", to: "."

            // Copy VERSION info into prisitine folder
            INFRAPOOL_EXECUTORV2_AGENT_0.agentSh "cp ../VERSION ./VERSION"

            // Create release artifacts without releasing to Github
            INFRAPOOL_EXECUTORV2_AGENT_0.agentSh "./bin/build_release --skip=validate --clean"

            // Build container images
            INFRAPOOL_EXECUTORV2_AGENT_0.agentSh "./bin/build"

            // Archive release artifacts
            INFRAPOOL_EXECUTORV2_AGENT_0.agentArchiveArtifacts artifacts: 'dist/goreleaser/'
          }
        }
      }
    }

    // Publish container images to internal registry. Need to push before we do security scans
    // since the Snyk scans pull from artifactory on a seprate executor node
    stage('Push images to internal registry') {
      steps {
        script {
          INFRAPOOL_EXECUTORV2_AGENT_0.agentSh './bin/publish --internal'
        }
      }
    }

    stage('Scan Docker Image') {
      steps {
        script {
          VERSION = INFRAPOOL_EXECUTORV2_AGENT_0.agentSh(returnStdout: true, script: 'cat VERSION')
        }
        runSecurityScans(INFRAPOOL_EXECUTORV2_AGENT_0,
          image: "registry.tld/${containerImageWithTag(INFRAPOOL_EXECUTORV2_AGENT_0)}",
          buildMode: params.MODE,
          branch: env.BRANCH_NAME)
      }
    }

    stage('Release') {
      when {
        expression {
          MODE == "RELEASE"
        }
      }
      steps {
        script {
          release(INFRAPOOL_EXECUTORV2_AGENT_0) { billOfMaterialsDirectory, assetDirectory, toolsDirectory ->
            // Publish release artifacts to all the appropriate locations
            // Copy any artifacts to assetDirectory to attach them to the Github release

            // Copy assets to be published in Github release.
            INFRAPOOL_EXECUTORV2_AGENT_0.agentSh "${toolsDirectory}/bin/copy_goreleaser_artifacts ${assetDirectory}"

            // Create Go application SBOM using the go.mod version for the golang container image
            INFRAPOOL_EXECUTORV2_AGENT_0.agentSh """export PATH="${toolsDirectory}/bin:${PATH}" && go-bom --tools "${toolsDirectory}" --go-mod ./go.mod --image "golang" --main "cmd/sidecar-injector" --output "${billOfMaterialsDirectory}/go-app-bom.json" """
            // Create Go module SBOM
            INFRAPOOL_EXECUTORV2_AGENT_0.agentSh """export PATH="${toolsDirectory}/bin:${PATH}" && go-bom --tools "${toolsDirectory}" --go-mod ./go.mod --image "golang" --output "${billOfMaterialsDirectory}/go-mod-bom.json" """
          }
        }
      }
    }
  }

  post {
    always {
      script {
        releaseInfraPoolAgent(".infrapool/release_agents")

        // Resolve ownership issue before running infra post hook
        sh 'git config --global --add safe.directory ${PWD}'
        infraPostHook()
      }
    }
  }
}

def gitCommit(infrapool) {
  infrapool.agentSh(
    returnStdout: true,
    script: 'source ./bin/build_utils && echo "$(git_commit)"'
  )
}

def containerImageWithTag(infrapool) {
  infrapool.agentSh(
    returnStdout: true,
    script: 'source ./bin/build_utils && echo "sidecar-injector:$(project_version_with_commit)"'
  )
}
