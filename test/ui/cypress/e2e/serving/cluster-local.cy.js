import ShowcaseKservice from '../../code/knative/serving/showcase'
import OpenshiftConsole from '../../code/openshift/openshiftConsole'

describe('OCP UI for Serverless Serving', () => {

  const openshiftConsole = new OpenshiftConsole()
  const showcaseKsvc = new ShowcaseKservice({
    clusterLocal: true,
    namespace:    'test-cluster-local'
  })

  it('can deploy a cluster-local service', () => {
    openshiftConsole.login()
    showcaseKsvc.removeApp()
    showcaseKsvc.deployImage()
    showcaseKsvc.url().and('include', 'cluster.local')
  })
})
