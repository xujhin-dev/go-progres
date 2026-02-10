package strategy

type PaymentStrategy interface {
	// Pay 发起支付，返回支付参数（如 URL、JSON 串）
	Pay(orderNo string, amount float64, subject string) (string, error)
	
	// Notify 处理回调通知，返回解析后的订单号、金额、支付状态
	Notify(params interface{}) (string, float64, bool, error)
}
