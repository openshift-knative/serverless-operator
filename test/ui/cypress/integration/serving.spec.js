describe('OCP UI for Serverless', () => {

  class ShowcaseKservice {
    constructor(ops = {}) {
      this.app = ops.app || 'demoapp'
      this.name = ops.name || 'showcase'
      this.namespace = ops.namespace || Cypress.env('TEST_NAMESPACE')
      this.image = ops.image || {
        regular: 'quay.io/cardil/knative-serving-showcase:2-send-event',
        updated: 'quay.io/cardil/knative-serving-showcase-js'
      }
    }

    makeRequest() {
      cy.get('a.co-external-link')
        .last()
        .scrollIntoView()
        .should('have.attr', 'href')
        .and('include', 'showcase')
        .then((href) => {
          const req = {
            method: 'OPTIONS',
            url: href,
            retryOnStatusCodeFailure: true,
            failOnStatusCode: true
          }
          cy.request(req).then((response) => {
            expect(response.status).to.eq(200)
            expect(response.body).to.have.property('version')
            expect(JSON.stringify(response.body)).to.include('knative-serving-showcase')
          })
        })
    }

    checkScale(scale) {
      const selector = 'div.pf-topology-container__with-sidebar ' +
        'div.odc-revision-deployment-list__pod svg tspan'
      cy.get(selector)
        .invoke('text')
        .should((text) => {
          expect(text).to.eq(`${scale}`)
        })
    }

    deployImage(kind = 'regular') {
      cy.visit(`/add/ns/${showcaseKsvc.namespace}`)
      cy.contains('Knative Channel')
      cy.contains('Event Source')
      cy.visit(`/deploy-image/ns/${showcaseKsvc.namespace}`)
      cy.get('input[name=searchTerm]')
        .type(showcaseKsvc.image[kind])
      cy.contains('Validated')
      cy.get('input#form-radiobutton-resources-knative-field').check()
      cy.get('input#form-checkbox-route-create-field').check()
      cy.get('input#form-input-application-name-field')
        .clear()
        .type(showcaseKsvc.app)
      cy.get('input#form-input-name-field')
        .clear()
        .type(showcaseKsvc.name)
      cy.get('button[type=submit]').click()
      cy.url().should('include', `/topology/ns/${showcaseKsvc.namespace}`)
      cy.contains(showcaseKsvc.app)
    }

    removeApp() {
      cy.visit('/dev-monitoring')
      cy.visit(`/topology/ns/${showcaseKsvc.namespace}/list`)
      cy.get('div.pf-topology-content')
        .contains(showcaseKsvc.app).click()
      cy.contains('Actions').click()
      cy.contains('Delete Application')
        .should('not.have.class', 'pf-m-disabled')
        .click()
      cy.get('input#form-input-resourceName-field')
        .type(showcaseKsvc.app)
      cy.get('button#confirm-action.pf-c-button.pf-m-danger').click()
      cy.contains('No resources found')
    }

    showServiceDetails() {
      cy.visit(`/topology/ns/${showcaseKsvc.namespace}/list`)
      cy.get('div.pf-topology-content')
        .contains(showcaseKsvc.name).click()
      cy.contains('Location:')
        .scrollIntoView()
    }
  }

  const showcaseKsvc = new ShowcaseKservice()

  it('can deploy kservice and scale it', () => {
    describe('with authenticated via Web Console', () => {
      cy.login()
    })
    describe('deploy kservice from image', () => {
      showcaseKsvc.deployImage()
    })
    describe('check automatic scaling of kservice', () => {
      showcaseKsvc.showServiceDetails()
      showcaseKsvc.makeRequest()
      showcaseKsvc.checkScale(1)
      cy.wait(60_000) // 60sec.

      showcaseKsvc.showServiceDetails()
      cy.contains('All Revisions are autoscaled to 0')
      showcaseKsvc.checkScale(0)
      showcaseKsvc.makeRequest()
      
      showcaseKsvc.showServiceDetails()
      showcaseKsvc.checkScale(1)
    })
    describe('remove kservice', () => {
      showcaseKsvc.removeApp()
    })
  })

  it('can route traffic to multiple revisions', () => {
    describe('with authenticated via Web Console', () => {
      cy.login()
    })
    describe('deploy kservice from image', () => {
      showcaseKsvc.deployImage()
    })
    describe('add two revisions to traffic distribution', () => {
      cy.visit(`/topology/ns/${showcaseKsvc.namespace}/list`)
      cy.get('div.pf-topology-content')
        .contains(showcaseKsvc.name).click()
      cy.contains('Actions').click()
      cy.contains(`Edit ${showcaseKsvc.name}`).click()
      cy.get('input[name=searchTerm]')
        .clear()
        .type(showcaseKsvc.image.updated)
      cy.contains('Validated')
      cy.get('button[type=submit]').click()
      cy.url().should('include', showcaseKsvc.namespace)
      cy.contains(showcaseKsvc.app)
      cy.visit(`/topology/ns/${showcaseKsvc.namespace}/list`)
      cy.get('div.pf-topology-content')
        .contains(showcaseKsvc.name).click()
      cy.contains('Set traffic distribution', { matchCase: false }).click()
      cy.get('input[name="trafficSplitting.0.percent"]')
        .clear()
        .type('51')
      cy.get('input[name="trafficSplitting.0.tag"]')
        .type('v2')
      cy.contains('Add Revision').click()
      cy.get('input[name="trafficSplitting.1.percent"]')
        .type('49')
      cy.get('input[name="trafficSplitting.1.tag"]')
        .type('v1')
      cy.contains('Select a Revision', { matchCase: false }).click()
      cy.get('ul.pf-c-dropdown__menu button').click()
      cy.get('button[type=submit]').click()
      cy.contains('51%')
      cy.contains('49%')
    })
    describe('check traffic distribution works', () => {
      cy.contains('Location:')
      for (let i = 0; i < 8; i++) {
        showcaseKsvc.makeRequest()
      }
    })
    describe('remove kservice', () => {
      showcaseKsvc.removeApp()
    })
  })
})
