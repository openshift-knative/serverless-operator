import Environment from '../../environment'
import Kubernetes from '../../kubernetes'
import OpenshiftConsole from '../../openshift/openshiftConsole'

const environment = new Environment()
const openshiftConsole = new OpenshiftConsole()

class ShowcaseKservice {

  constructor(ops = {}) {
    this.app = ops.app || 'demoapp'
    this.name = ops.name || 'showcase'
    this.namespace = ops.namespace || Cypress.env('TEST_NAMESPACE')
    this.clusterLocal = ops.clusterLocal || false
    this.image = ops.image || {
      regular: 'quay.io/openshift-knative/showcase',
      updated: 'quay.io/openshift-knative/showcase:js'
    }
  }

  /**
   * Gets the URL of the deployed kservice.
   * @returns {Cypress.Chainable<URL>} - the URL of the kservice
   */
  url() {
    if (this.clusterLocal) {
      let selector = '.overview__sidebar-pane .pf-v5-c-clipboard-copy input[type=text]'
      if (environment.ocpVersion().satisfies('<=4.14')) {
        selector = '.overview__sidebar-pane .pf-c-clipboard-copy input[type=text]'
      }
      return cy.get(selector)
        .last()
        .scrollIntoView()
        .should('have.attr', 'value')
        .and('include', 'showcase')
    }
    let selector = '.co-external-link--block a'
    if (environment.ocpVersion().satisfies('<=4.13')) {
      selector = 'a.co-external-link'
    }
    return cy.get(selector)
      .last()
      .scrollIntoView()
      .should('have.attr', 'href')
      .and('include', 'showcase')
  }

  makeRequest(baseUrl) {
    const req = {
      method: 'GET',
      url: baseUrl,
      retryOnStatusCodeFailure: true,
      failOnStatusCode: true,
      headers: {'user-agent': `Cypress/${Cypress.version}`}
    }
    cy.request(req).then((response) => {
      expect(response.status).to.eq(200)
      expect(response.headers)
        .to.have.property('content-type')
        .that.matches(/^application\/json(?:;.+)?$/)
      expect(response.headers).to.have.property('x-version')
      expect(response.headers).to.have.property('x-config')
      expect(response.headers).to.have.property('server')
      expect(response.body).to.have.property('artifact', 'knative-showcase')
      expect(response.body).to.have.property('greeting', 'Welcome')
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

  deployImage({kind = 'regular'} = {}) {
    cy.log(`Deploy kservice ${kind}${this.clusterLocal ? ', cluster-local' : ''} from image`)
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
    cy.get('input#form-checkbox-route-create-field')
      .scrollIntoView()
      .check()
    if (this.clusterLocal) {
      cy.get('input#form-checkbox-route-create-field')
        .scrollIntoView()
        .uncheck()
    }
    // FIXME: Remove after https://issues.redhat.com/browse/OCPBUGS-38680 is fixed.
    if (environment.ocpVersion().satisfies('>=4.17')) {
      const resourceSelect =
        cy.get('button#form-select-input-resources-field')

      resourceSelect
        .scrollIntoView()
        .click()
      resourceSelect.siblings('ul[role=listbox]')
        .get('li#select-option-resources-knative button')
        .click()
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

  // FIXME: Delete after https://issues.redhat.com/browse/OCPBUGS-6685 is fixed.
  deleteAppGroupViaKubectl() {
    cy.log(`Delete app group ${this.app} using CLI`)
    return new Cypress.Promise((resolve, _) => {
      const cmd = `kubectl delete all -l app.kubernetes.io/part-of=${this.app} -n ${this.namespace} -o name`
      cy.exec(cmd).then((result) => {
        cy.log(result.stdout)
        let out = result.stdout.trim()
        resolve(out.length > 0)
      })
    })
  }

  removeApp() {
    const k8s = new Kubernetes()
    k8s.ensureNamespaceExists(this.namespace)
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
      .get('li[data-test-action="Delete application"] button')
      .should('be.visible')
      .should('not.be.disabled')
      .click()
    cy.get('input#form-input-resourceName-field')
      .type(this.app)
    cy.get('.modal-content button#confirm-action.pf-m-danger').click()
    // FIXME: https://issues.redhat.com/browse/OCPBUGS-6685
    //        Removal of the app sometimes leaves a image stream, making the UI
    //        stale.
    this.deleteAppGroupViaKubectl().then((_deleted) => {
      cy.get('div.pf-topology-content')
        .contains('No resources found')
    })
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
    return `/topology/ns/${this.namespace}?view=${kind}`
  }
}

export default ShowcaseKservice
