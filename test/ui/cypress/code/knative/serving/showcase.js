import Environment from '../../environment'
import OpenshiftConsole from '../../openshift/openshiftConsole'

const environment = new Environment()
const openshiftConsole = new OpenshiftConsole()

class ShowcaseKservice {

  constructor(ops = {}) {
    this.app = ops.app || 'demoapp'
    this.name = ops.name || 'showcase'
    this.namespace = ops.namespace || Cypress.env('TEST_NAMESPACE')
    this.image = ops.image || {
      // TODO(ksuszyns): SRVCOM-1235 donate those apps to openshift-knative
      regular: 'quay.io/cardil/knative-serving-showcase:2-send-event',
      updated: 'quay.io/cardil/knative-serving-showcase-js'
    }
  }

  url() {
    return cy.get('a.co-external-link')
      .last()
      .scrollIntoView()
      .should('have.attr', 'href')
      .and('include', 'showcase')
  }

  makeRequest(baseUrl) {
    const req = {
      method: 'OPTIONS',
      url: baseUrl,
      retryOnStatusCodeFailure: true,
      failOnStatusCode: true
    }
    cy.request(req).then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body).to.have.property('version')
      expect(JSON.stringify(response.body)).to.include('knative-serving-showcase')
    })
  }

  checkScale(scale) {
    cy.log(`check scale of ${this.name} to ${scale}`)
    const selector = 'div.odc-revision-deployment-list__pod svg tspan'
    const timeout = Cypress.config().defaultCommandTimeout
    try {
      // TODO: Remove the increased timeout when https://issues.redhat.com/browse/ODC-5685 is fixed.
      Cypress.config('defaultCommandTimeout', 300_000)
      cy.get(selector)
        .invoke('text')
        .should((text) => {
          expect(text).to.eq(`${scale}`)
        })
    } finally {
      Cypress.config('defaultCommandTimeout', timeout)
    }
  }

  deployImage({kind = 'regular', clusterLocal = false} = {}) {
    cy.log(`Deploy kservice ${kind}${clusterLocal ? ', cluster-local' : ''} from image`)
    cy.visit(`/deploy-image/ns/${this.namespace}`)
    cy.get('input[name=searchTerm]')
      .type(this.image[kind])
    cy.contains('Validated')
    cy.get('input#form-input-application-name-field')
      .scrollIntoView()
      .clear()
      .type(this.app)
    cy.get('input#form-input-name-field')
      .scrollIntoView()
      .clear()
      .type(this.name)
    cy.get('input#form-radiobutton-resources-knative-field')
      .scrollIntoView()
      .check()
    cy.get('input#form-checkbox-route-create-field')
      .scrollIntoView()
      .check()
    if (clusterLocal) {
      cy.get('input#form-checkbox-route-create-field')
        .scrollIntoView()
        .uncheck()
    }
    cy.get('button[type=submit]')
      .scrollIntoView()
      .click()
    cy.url().should('include', `/topology/ns/${this.namespace}`)
    cy.visit(this.topologyUrl())
    cy.get('div.pf-topology-content').contains(this.name).click()
    // Make sure the app is running before proceeding.
    cy.contains('Running')
  }

  isServiceDeployed() {
    return new Cypress.Promise((resolve, _) => {
      const cmd = `kubectl get all -l app.kubernetes.io/part-of=${this.app} -n ${this.namespace} -o name`
      cy.exec(cmd).then((result) => {
        cy.log(result.stdout)
        let out = result.stdout.trim()
        resolve(out.length > 0)
      })
    })
  }

  removeApp() {
    this.isServiceDeployed().then((deployed) => {
      if (deployed) {
        cy.log("Service is deployed. Removing it.")
        this.doRemoveApp()
      } else {
        cy.log("Service isn't deployed, skipping removal.")
      }
    })
  }

  doRemoveApp() {
    cy.visit(this.topologyUrl())
    openshiftConsole.closeSidebar()
    cy.get('div.pf-topology-content')
      .contains(this.app).click()
    const selectors = openshiftConsole.sidebarSelectors()
    const drawer = cy.get(selectors.drawer)
    drawer
      .contains('Actions')
      .should('be.visible')
      .should('not.be.disabled')
      .click()
    drawer
      .get(selectors.deleteApplicationBtn)
      .should('be.visible')
      .should('not.be.disabled')
      .click()
    cy.get('input#form-input-resourceName-field')
      .type(this.app)
    cy.get('button#confirm-action.pf-c-button.pf-m-danger').click()
    cy.contains('No resources found')
  }

  showServiceDetails(scrollTo = 'Location:') {
    cy.log('Show service details')
    cy.visit(this.topologyUrl())
    openshiftConsole.closeSidebar()
    cy.get('div.pf-topology-content')
      .contains(this.name)
      .click() // opens the sidebar
    cy.contains(scrollTo)
      .scrollIntoView()
  }

  topologyUrl(kind = 'list') {
    const ver = environment.ocpVersion()
    if (ver.satisfies('>=4.9')) {
      return `/topology/ns/${this.namespace}?view=${kind}`
    } else {
      return `/topology/ns/${this.namespace}/${kind}`
    }
  }
}

export default ShowcaseKservice
