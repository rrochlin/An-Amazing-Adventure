document.addEventListener('DOMContentLoaded', (event) => {
	console.log('DOM fully loaded and parsed');
	console.log("calling maze.js")
	// first we need to create a stage
	var stage = new Konva.Stage({
		container: 'container', // id of container <div>
		width: 1000,
		height: 1000,
	});

	// then create layer
	var layer = new Konva.Layer();

	const dim = 40
	const inc = 1000 / dim
	const maze = []

	for (var i = 0; i <= dim; i++) {
		maze[i] = []
		for (var j = 0; j <= dim; j++) {
			const rect1 = new Konva.Rect({
				x: i * inc,
				y: j * inc,
				width: inc,
				height: inc,
				fill: 'green',
				stroke: 'black',
				strokeWidth: 1
			});
			maze[i].push(rect1)
			layer.add(rect1);
		}
	}

	// add the layer to the stage
	stage.add(layer);
	function sleep(ms) {
		return new Promise(resolve => setTimeout(resolve, ms));
	}
	async function redraw() {
		while (true) {
			for (const color of extendedColors) {
				for (var i = 0; i <= dim; i++) {
					for (var j = 0; j <= dim; j++) {
						maze[j][i].fill(color)
						await sleep(10)

					}
				}
			}

		}
	}
	const extendedColors = [
		'#1f77b4', // Muted Blue
		'#ff7f0e', // Safety Orange
		'#2ca02c', // Cooked Asparagus Green
		'#d62728', // Brick Red
		'#9467bd', // Muted Purple
		'#8c564b', // Cast Iron Brown
		'#e377c2', // Raspberry Yogurt Pink
		'#7f7f7f', // Middle Gray
		'#bcbd22', // Curry Yellow-Green
		'#17becf', // Blue-Teal
		'#aec7e8', // Light Blue
		'#ffbb78', // Peach
		'#98df8a', // Light Green
		'#ff9896', // Light Red
		'#c5b0d5', // Light Purple
		'#c49c94', // Light Brown
		'#f7b6d2', // Light Pink
		'#c7c7c7', // Light Gray
		'#dbdb8d', // Light Yellow-Green
		'#9edae5'  // Light Blue-Teal
	];

	redraw()
});
