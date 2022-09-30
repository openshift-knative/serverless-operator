import ShowcaseKservice from '../../code/knative/serving/showcase'
import OpenshiftConsole from '../../code/openshift/openshiftConsole'

describe('OCP UI for Serverless Serving', () => {

  const openshiftConsole = new OpenshiftConsole()
  const showcaseKsvc = new ShowcaseKservice()

  it('can deploy kservice and scale it', () => {
    openshiftConsole.login()
    showcaseKsvc.removeApp()

    showcaseKsvc.deployImage()

    cy.log('check automatic scaling of kservice')
    showcaseKsvc.showServiceDetails()
    showcaseKsvc.url().then((url) => {
      showcaseKsvc.makeRequest(url)
      showcaseKsvc.checkScale(1)
      cy.wait(60_000) // 60sec.

      showcaseKsvc.showServiceDetails()
      cy.contains('All Revisions are autoscaled to 0')
      showcaseKsvc.checkScale(0)
      showcaseKsvc.makeRequest(url)

      showcaseKsvc.showServiceDetails()
      showcaseKsvc.checkScale(1)
    })
  })
})
