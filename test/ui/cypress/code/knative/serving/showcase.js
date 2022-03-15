import Environment from "../../environment";

const environment = new Environment()

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
    cy.visit(`/add/ns/${this.namespace}`)
    cy.contains('Knative Channel')
    cy.contains('Event Source')
    cy.visit(`/deploy-image/ns/${this.namespace}`)
    cy.get('input[name=searchTerm]')
      .type(this.image[kind])
    cy.contains('Validated')
    cy.get('input#form-radiobutton-resources-knative-field').check()
    cy.get('input#form-checkbox-route-create-field').check()
    cy.get('input#form-input-application-name-field')
      .clear()
      .type(this.app)
    cy.get('input#form-input-name-field')
      .clear()
      .type(this.name)
    if (clusterLocal) {
      cy.get('input#form-checkbox-route-create-field')
        .uncheck()
    }
    cy.get('button[type=submit]').click()
    cy.url().should('include', `/topology/ns/${this.namespace}`)
    cy.visit(this.topologyUrl())
    cy.get('div.pf-topology-content').contains(this.name).click()
    // Make sure the app is running before proceeding.
    cy.contains('Running')
  }

  isServiceDeployed() {
    return new Cypress.Promise((resolve, _) => {
      const cmd = `kubectl get all -l app.kubernetes.io/part-of=${this.app} -n ${this.namespace}`
      cy.exec(cmd, { failOnNonZeroExit: false }).then(result => {
        resolve(result.code === 0)
      })
    })
  }

  removeApp() {
    this.isServiceDeployed().then(deployed => {
      if (deployed) {
        this.doRemoveApp()
      } else {
        cy.log("Service isn't deployed, skipping removal.")
      }
    })
  }

  doRemoveApp() {
    const env = new Environment()
    const rng = env.random().next()
    const self = this
    const ways = [
      () => { return self.removeAppViaKubectl() },
      // FIXME: This do not work on OCP 4.10+ See: https://issues.redhat.com/browse/OCPBUGSM-41912
      // () => { return self.removeAppViaUI() },
    ]
    const idx = Math.floor(rng * ways.length)
    const way = ways[idx]
    return way()
  }

  // FIXME: This do not work on OCP 4.10+ See: https://issues.redhat.com/browse/OCPBUGSM-41912
  removeAppViaUI() {
    cy.visit(this.topologyUrl())
    cy.get('div.pf-topology-content')
      .contains(this.app).click()
    cy.contains('Actions').click()
    cy.contains('Delete Application')
      .should('not.have.class', 'pf-m-disabled')
      .click()
    cy.get('input#form-input-resourceName-field')
      .type(this.app)
    cy.get('button#confirm-action.pf-c-button.pf-m-danger').click()
    cy.contains('No resources found')
  }

  removeAppViaKubectl() {
    const cmd = `kubectl delete all -l app.kubernetes.io/part-of=${this.app} -n ${this.namespace}`
    cy.exec(cmd).then(result => {
      if (result.code !== 0) {
        throw new Error(`Command failed with code ${result.code}: \`${cmd}\`.\nStdout: ${result.stdout}\nStderr: ${result.stderr}`)
      }
    })
  }

  showServiceDetails(scrollTo = 'Location:') {
    cy.visit(this.topologyUrl())
    cy.get('div.pf-topology-content')
      .get('#serving\\.knative\\.dev\\~v1\\~Service_label')
      .click() // closes the sidebar if open
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
