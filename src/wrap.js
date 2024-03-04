(function() {

    const traverse = function (d, callback) {
        // z = document.createElement('div')
        // z.childNodes
        for (c of d.childNodes) {
            callback(c);
            traverse(c, callback);
        }
    };

    const wrap = function (parent, t) {

        if ('string' == typeof parent) {
            parent = document.querySelector(parent);
        }

        let div = document.createElement('div');
        div.innerHTML = t;

        let pattern = /{{([^}]+)}}/mg;

        let values = {};
        let updates = {};
        let events = {};

        traverse(div, function(node) {
            if (node.nodeType === 3) {
                // text node, look for replacements
                let text = node.textContent;
                let parts = [];
                let placeholders = {};

                let it = text.matchAll(pattern);
                let lastIndex = 0;
                for (;;) {
                    let match = it.next();
                    if (match.done) break;

                    parts.push(text.substring(lastIndex, match.value.index))
                    let placeholder = match.value[1].trim();
                    placeholders[parts.length] = placeholder;
                    parts.push(placeholder);
                    values[placeholder] = '';
                    updates[placeholder] = updates[placeholder] || [];
                    updates[placeholder].push({
                        node: node,
                        parts: parts,
                        placeholders: placeholders,
                    });
                    lastIndex = match.value.index + match.value[0].length;
                }
                parts.push(text.substring(lastIndex))
            }
            if (node.nodeType === 1) {
                // dom node, look for attributes
                for (let attribute of node.attributes) {

                    if (attribute.name['0'] === '@') {
                        let name = attribute.name.substring(1);
                        events[name] = events[name] || [];
                        events[name].push(node);
                        continue;
                    }

                    let text = attribute.value;
                    let parts = [];
                    let placeholders = {};

                    let it = text.matchAll(pattern);
                    let lastIndex = 0;
                    for (;;) {
                        let match = it.next();
                        if (match.done) break;

                        parts.push(text.substring(lastIndex, match.value.index))
                        let placeholder = match.value[1].trim();
                        placeholders[parts.length] = placeholder;
                        parts.push(placeholder);
                        values[placeholder] = '';
                        updates[placeholder] = updates[placeholder] || [];
                        updates[placeholder].push({
                            node: attribute,
                            parts: parts,
                            placeholders: placeholders,
                        });
                        lastIndex = match.value.index + match.value[0].length;
                    }
                    parts.push(text.substring(lastIndex))
                }
            }
        });

        // move nodes to parent
        let refs = [];
        for (node of div.childNodes) {
            refs.push(node);
        }
        refs.forEach(node => parent.append(node));

        return {
            setValues(o) {
                for (let k in o) {
                    this.setValue(k, o[k])
                }
            },
            setValue(key, value){
                let update = updates[key];
                if (!update) return;

                values[key] = value;

                update.forEach(u => {
                    let text = '';
                    for (let i in u.parts) {
                        if (u.placeholders[i]) {
                            text += values[u.placeholders[i]];
                        } else {
                            text += u.parts[i]
                        }
                    }
                    u.node.textContent = text;
                });

            },
            setEvent(name, type, callback) {
                let event = events[name];
                if (!event) return;
                for (let e of event) {
                    e.addEventListener(type, function (e) {
                        callback(e, values);
                    }, true);
                }
            },
        }
    };

    window.Wrap = wrap;
})();