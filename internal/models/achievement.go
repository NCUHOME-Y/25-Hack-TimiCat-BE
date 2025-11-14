package models

type Achievement struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Content     string `json:"content"`
	Subtitle    string `json:"subtitle"`
	Achievement string `json:"achievement"`
	Threshold   int64  `json:"requirement"`
	Unlocked    bool   `json:"unlocked"`
}

var Achievements = []Achievement{
	{
		ID:          1,
		Name:        "初遇",
		Content:     "世界很小，小到在雪落之时，你遇到了一只属于自己的小猫。",
		Subtitle:    "第一次见到小猫",
		Achievement: "人，\n你等了咪很多个冬天吗？",
		Threshold:   1, // 1秒测试

	},
	{
		ID:          2,
		Name:        "无声告白",
		Content:     "专注的力量能让时间变得有意义。",
		Subtitle:    "专注时长累计达到 5h 20min",
		Achievement: "人，\n爱咪，或者不爱咪，\n咪都在这里。",
		Threshold:   5*3600 + 20*60, // 5小时20分

	},
	{
		ID:          3,
		Name:        "答案",
		Content:     "每一次坚持都是对自己的承诺。",
		Subtitle:    "专注时长累计达到 24h",
		Achievement: "人，\n春天远远的，你呢？",
		Threshold:   24 * 3600, //24小时

	},
	{
		ID:          4,
		Name:        "猫岛",
		Content:     "时间会记住你所有的努力。",
		Subtitle:    "专注时长累计达到 36h",
		Achievement: "人，\n你的小岛上，\n只有我一只咪吗？",
		Threshold:   36 * 3600, //36小时

	},
}
