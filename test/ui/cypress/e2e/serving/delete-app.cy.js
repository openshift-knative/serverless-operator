import Environment from '../../code/environment'
import ShowcaseKservice from '../../code/knative/serving/showcase'
import OpenshiftConsole from '../../code/openshift/openshiftConsole'

describe('OCP UI for Serverless Serving', () => {
  const environment = new Environment()
  const openshiftConsole = new OpenshiftConsole()
  const showcaseKsvc = new ShowcaseKservice({
    namespace: 'test-delete-app'
  })

  it('can delete an app with Knative service', () => {
    openshiftConsole.login()
    showcaseKsvc.removeApp()
    showcaseKsvc.deployImage()
    showcaseKsvc.removeApp()
    showcaseKsvc.deployImage()
    showcaseKsvc.url().then((url) => {
      showcaseKsvc.makeRequest(url)
    })
  })
})
