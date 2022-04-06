import Environment from "../code/environment";
import ShowcaseKservice from "../code/knative/serving/showcase";
import OpenshiftConsole from "../code/openshift/openshiftConsole";

describe('OCP UI for Serverless', () => {

  const environment = new Environment()
  const openshiftConsole = new OpenshiftConsole()
  const showcaseKsvc = new ShowcaseKservice()

  beforeEach(() => {
    describe('with authenticated via Web Console', () => {
      openshiftConsole.login()
    })
    describe('remove app', () => {
      showcaseKsvc.removeApp()
    })
  })

  it('can deploy kservice and scale it', () => {
    describe('deploy kservice from image', () => {
      showcaseKsvc.deployImage()
    })
    describe('check automatic scaling of kservice', () => {
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

  it('can route traffic to multiple revisions', () => {
    describe('deploy kservice from image', () => {
      showcaseKsvc.deployImage()
    })
    describe('add two revisions to traffic distribution', () => {
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
      cy.get('ul.pf-c-dropdown__menu button').click()
      cy.get('button[type=submit]').click()

      // FIXME: Remove after fixing https://issues.redhat.com/browse/OCPBUGSM-41966
      showcaseKsvc.showServiceDetails()

      cy.contains('51%')
      cy.contains('49%')
    })
    describe('check traffic distribution works', () => {
      cy.contains('Location:')
      showcaseKsvc.url().then((url) => {
        for (let i = 0; i < 8; i++) {
          showcaseKsvc.makeRequest(url)
        }
      })
    })
  })

  it('can deploy a cluster-local service', () => {
    const range = '>=4.8 || ~4.7.18 || ~4.6.39'
    cy.onlyOn(environment.ocpVersion().satisfies(range))

    describe('deploy kservice from image', () => {
      showcaseKsvc.deployImage({clusterLocal: true})
    })
    describe('check if URL contains cluster.local', () => {
      showcaseKsvc.url()
        .and('include', 'cluster.local')
    })
  })
})
