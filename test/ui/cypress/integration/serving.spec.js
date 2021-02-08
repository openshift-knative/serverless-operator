describe('OCP UI for Serverless', () => {

  class ShowcaseKservice {
    constructor(ops = {}) {
      this.app = ops.app || 'demoapp'
      this.name = ops.name || 'showcase'
      this.namespace = ops.namespace || 'default'
      this.image = ops.image ||
        'quay.io/cardil/knative-serving-showcase:2-send-event'
    }

    makeRequest() {
      cy.get('a.co-external-link')
      .scrollIntoView()
      .should('have.attr', 'href')
      .and('include', 'showcase')
      .then((href) => {
        const req = {
          method: 'OPTIONS',
          url: href,
          retryOnStatusCodeFailure: true
        }
        cy.request(req).then((response) => {
          expect(response.body).to.have.property('version')
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
  }

  const showcaseKsvc = new ShowcaseKservice()
  

  it('can deploy kservice and scale it', () => {
    describe('with authenticated via Web Console', () => {
      cy.login()
    })
    describe('deploy kservice from image', () => {
      cy.visit(`/add/ns/${showcaseKsvc.namespace}`)
      cy.contains('Knative Channel')
      cy.contains('Event Source')
      cy.visit(`/deploy-image/ns/${showcaseKsvc.namespace}`)
      cy.get('input[name=searchTerm]')
        .type(showcaseKsvc.image)
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
    })
    describe('check availibility of kservice', () => {
      cy.contains(showcaseKsvc.app)
      cy.visit(`/topology/ns/${showcaseKsvc.namespace}/list`)
      cy.get('div.pf-topology-content')
        .contains(showcaseKsvc.name).click()
      cy.contains('Location:')
      cy.contains('Running')
      showcaseKsvc.makeRequest()
      showcaseKsvc.checkScale(1)
      cy.wait(60_000) // 60sec.
      showcaseKsvc.checkScale(0)
      showcaseKsvc.makeRequest()
      showcaseKsvc.checkScale(1)
    })
    describe('remove kservice', () => {
      cy.visit(`/topology/ns/${showcaseKsvc.namespace}/list`)
      cy.get('div.pf-topology-content')
        .contains(showcaseKsvc.app).click()
      cy.contains('Actions').click()
      cy.contains('Delete Application').click()
      cy.get('input#form-input-resourceName-field')
        .type(showcaseKsvc.app)
      cy.get('button#confirm-action.pf-c-button.pf-m-danger').click()
      cy.contains('No resources found')
    })
  })
})
