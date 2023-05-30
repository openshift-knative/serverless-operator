import Environment from '../../code/environment'
import ShowcaseKservice from '../../code/knative/serving/showcase'
import OpenshiftConsole from '../../code/openshift/openshiftConsole'

describe('OCP UI for Serverless Serving', () => {

  const environment = new Environment()
  const openshiftConsole = new OpenshiftConsole()
  const showcaseKsvc = new ShowcaseKservice({
    clusterLocal: true,
  })

  it('can deploy a cluster-local service', () => {
    const range = '>=4.8 || ~4.7.18 || ~4.6.39'
    cy.onlyOn(environment.ocpVersion().satisfies(range))

    openshiftConsole.login()
    showcaseKsvc.removeApp()
    showcaseKsvc.deployImage()
    showcaseKsvc.url().and('include', 'cluster.local')
  })
})
