import ShowcaseKservice from '../../code/knative/serving/showcase'
import OpenshiftConsole from '../../code/openshift/openshiftConsole'
import Environment from '../../code/environment'

describe('OCP UI for Serverless Serving', () => {

  const openshiftConsole = new OpenshiftConsole()
  const showcaseKsvc = new ShowcaseKservice({
    namespace: 'test-multiple-revisions'
  })
  const environment = new Environment()

  it('can route traffic to multiple revisions', () => {
    openshiftConsole.login()
    showcaseKsvc.removeApp()

    showcaseKsvc.deployImage()
    showcaseKsvc.showServiceDetails()

    cy.contains('Actions').click()
    cy.contains(`Edit ${showcaseKsvc.name}`).click()
    cy.get('input[name=searchTerm]')
      .clear()
      .type(showcaseKsvc.image.updated)
    cy.contains('Validated')
    cy.get('button[type=submit]').click()
    cy.url()
      .should('not.include', '/edit/')
      .should('include', showcaseKsvc.namespace)
    cy.contains(showcaseKsvc.app)
    showcaseKsvc.showServiceDetails()
    cy.contains('Set traffic distribution', {matchCase: false}).click()
    cy.get('input[name="trafficSplitting.0.percent"]')
      .clear()
      .type('51')
    cy.get('input[name="trafficSplitting.0.tag"]')
      .type('v2')
    cy.contains('Add Revision')
      .should('not.be.disabled')
      .click()
    cy.get('input[name="trafficSplitting.1.percent"]')
      .type('49')
    cy.get('input[name="trafficSplitting.1.tag"]')
      .type('v1')
    cy.contains('Select a Revision', {matchCase: false}).click()
    let selector = `ul.pf-v5-c-dropdown__menu button`
    if (environment.ocpVersion().satisfies('<=4.14')) {
      selector = `ul.pf-c-dropdown__menu button`
    }
    cy.get(selector).click()
    cy.get('button[type=submit]').click()

    cy.log('Verify traffic is routed to both v1 and v2')
    cy.contains('51%')
    cy.contains('49%')

    cy.log('check traffic distribution works')
    cy.contains('Location:')
    showcaseKsvc.url().then((url) => {
      for (let i = 0; i < 8; i++) {
        showcaseKsvc.makeRequest(url)
      }
    })

  })

})
