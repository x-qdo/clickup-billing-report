@Library('pipeline-common@master') _

def projectName = 'clickup-report'

properties(skyDeployer.wrapProperties())

withResultReporting(slackChannel: '#ci') {
  inDockerAgent([
    arch: 'arm64',
  ]) {
    checkout(scm)

    skyDeployer.deployOnCommentTrigger(
      kubernetesDeployment: projectName,
      lockGlobally: false,
      deployMap: [
        'beta': [
          'preCheck': false,
          'env': 'beta',
        ],
      ],
      checklistFor: { env ->
        def url = ""
        if (env.name == "beta") {
          url = "https://clickup-reports.beta.hdo.ee/"
        }
        [[
          name: 'OK?',
          description: "Are you feeling good about this change in ${env.name}? ${url}"
        ]]
      },
    )
  }
}
