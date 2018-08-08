package admission

import (
	"net/http"
	"github.com/golang/glog"
	"io/ioutil"
	"encoding/json"
	"k8s.io/api/admission/v1beta1"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const admissionLogTag = "[admission]"

type MachineAdmissionRequest struct {
	KubeClient *kubernetes.Clientset
}

func (mar MachineAdmissionRequest) HandleAdmission(w http.ResponseWriter, r *http.Request) {
	if mar.KubeClient == nil {
		glog.Errorf("KubeClient not set for MachineAdmissionRequest.")
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		glog.V(2).Infof("%s Cannot process request: Received non-JSON Content-Type", admissionLogTag)
	}

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.V(2).Infof("%s Unable to read from request body: %v", admissionLogTag, err)
		return
	}

	if !json.Valid(reqBody) {
		glog.V(2).Infof("%s Cannot process request: Request body is not a valid json!", admissionLogTag)
		return
	}

	ar := v1beta1.AdmissionReview{}
	if err := ar.Unmarshal(reqBody); err != nil {
		glog.V(2).Infof("%s Could not unmarshal admission request body: %v", admissionLogTag, err)
		return
	}

	ms := v1alpha1.Machine{}
	if err := ms.Unmarshal(reqBody); err != nil {
		glog.V(2).Infof("%s Could not unmarshal admission request body: %v", admissionLogTag, err)
		return
	}

	admissionResp := v1beta1.AdmissionResponse{}
	admissionResp.Allowed = true

	// VALIDATE
	mvr := ValidateMachine(machineSpec, mar.KubeClient)
	if mvr.ExitCode != OK {
		glog.V(2).Infof("%s Machine Admission was negative for '%s': %v", admissionLogTag, machineSpec.Name, mvr.Error)
		admissionResp.Allowed = false
		admissionResp.Result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: mvr.Error.Error(),
			Reason:  metav1.StatusReason(mvr.Error.Error()),
			Code:    400,
		}
	}

	response := v1beta1.AdmissionReview{
		Response: &admissionResp,
	}

	var resp []byte
	response.MarshalTo(resp)
	if _, err := w.Write(resp); err != nil {
		glog.Error(err)
	}

}
