import Environment from '../../code/environment'
import ShowcaseKservice from '../../code/knative/serving/showcase'
import OpenshiftConsole from '../../code/openshift/openshiftConsole'

describe('OCP UI for Serverless Eventing', () => {

  const environment = new Environment()
  const openshiftConsole = new OpenshiftConsole()
  const ns = Cypress.env('TEST_NAMESPACE')

  it('have Eventing bits to add', () => {
    openshiftConsole.login()
    cy.visit(`/add/ns/${ns}`)
    cy.contains('Event Source')
    if (environment.ocpVersion().satisfies('>=4.11')) {
      cy.contains('Event Sink')
    } else {
      cy.contains('Channel')
    }
  })
})
